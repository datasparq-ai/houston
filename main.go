package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"

	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/datasparq-ai/houston/client"
	"github.com/datasparq-ai/houston/database"
	"github.com/datasparq-ai/houston/mission"
	"github.com/datasparq-ai/houston/model"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/acme/autocert"
	// "golang.org/x/term"
)

// API is an instance of the Houston orchestration API
type API struct {
	db       database.Database // connection to a database instance, either Redis or 'local' held within a single object in Go
	router   *mux.Router       // the router that routes all requests to request handlers
	ws       chan message      // WebSocket channel. Any events sent
	config   Config            // configuration - see docs/config for full documentation
	protocol string            // this is set to either 'http' or 'https' depending on config.TLSConfig
}

// New creates the Houston API object.
// It will create or connect to a database depending on the settings in the config file.
// local db will only persist while program is running.
func New(configPath string) *API {
	initLog()

	recovering := false

	config := LoadConfig(configPath)

	var db database.Database
	// attempt to connect to redis - if not found then use local db
	db = database.NewRedisDatabase(config.Redis.Addr, config.Redis.Password, config.Redis.DB)
	err := db.Ping()
	switch e := err.(type) {
	case nil:
		msg := fmt.Sprintf("Connected to Redis Database at %v", config.Redis.Addr)
		log.Info(msg)
		if isTerminal {
			fmt.Println("🚨 " + msg)
		}

		// note: servers that don't have passwords can't recover - this is intentional to aid local development / unit tests
		_, passwordExists := db.Get("m", "p")
		_, saltExists := db.Get("m", "s")

		if passwordExists && saltExists {

			keys, _ := db.ListKeys()

			msg := fmt.Sprintf("Houston is recovering using existing settings, keys, and plans - found %v keys", len(keys))
			log.Info(msg)
			if isTerminal {
				fmt.Println("🔧 " + msg)
			}
			recovering = true
		}

	case *net.OpError:
		switch e.Err.(type) {
		case *os.SyscallError:
			// TODO: fail in production mode (and unittest mode)

			msg := fmt.Sprintf("Couldn't connect to Redis Database at %v.", config.Redis.Addr)
			log.Warn(msg)

			// if redis address is not the default then this is an error
			if config.Redis.Addr != "localhost:6379" {
				panic(msg)
			} else if isTerminal {
				fmt.Println("⚠️ " + msg + " Using in-memory database")
			}

			db = database.NewLocalDatabase()
		case *net.AddrError:
			log.Error("Do not add protocol to Redis.Addr")
			log.Error(err)
			panic(err) // this happens when user puts protocol in Redis.Addr
		default:
			log.Error(err)
			panic(err)
		}
	default:
		log.Error(err)
		panic(err)
	}

	// if non-default TLS cert is being used, or certs exist at the default location, try to use https
	protocol := "http"
	if config.TLS.Host != "" || config.TLS.CertFile != "cert.pem" || config.TLS.KeyFile != "key.pem" {
		protocol = "https"
	} else {
		if _, err := os.Stat(config.TLS.CertFile); err == nil {
			protocol = "https"
		}
		if _, err := os.Stat(config.TLS.KeyFile); err == nil {
			protocol = "https"
		}
	}

	a := API{db, nil, nil, config, protocol}

	log.Debugf("API will use the %s protocol", protocol)

	a.config.Password = strings.Trim(a.config.Password, " \n\t")
	if recovering {
		log.Info("Houston is recovering and password already exists, so config.Password will be ignored")
		password, _ := db.Get("m", "p") // this is already hashed
		salt, _ := db.Get("m", "s")
		if hashPassword(a.config.Password, salt) != password {
			log.Warn("Password provided via the config object or HOUSTON_PASSWORD environment variable doesn't match " +
				"the password found when recovering. This happens because the password has been changed since the config was set. " +
				"Ensure that you do not accidentally use the old password.")
		}
		a.config.Password = password
		a.config.Salt = salt
	} else if config.Password != "" {
		err := a.SetPassword(a.config.Password)
		if err != nil {
			if isTerminal {
				panic(err)
			}
			log.Fatal(err)
		}
		log.Debug("API password set successfully")
	} else {
		if protocol == "https" {
			// assume that https is being used because server is in production
			err := fmt.Errorf("It is not recommended to run Houston in production without setting a server password, as this allows anyone to create or delete API keys.")
			log.Error(err)
			panic(err)
		}
		log.Warn("API has no admin password")
	}

	a.initRouter()
	a.initDashboard()
	a.initWebSocket()
	return &a
}

func (a *API) SetPassword(password string) error {
	log.Info("Attempt made to set new password")

	if len(password) < 10 {
		log.Error("Failed to set new password")
		return fmt.Errorf("password provided is not long enough. Houston admin password must be at least 10 characters. Recommended length is 30")
	}
	if strings.ContainsAny(password, "\\ \t\n") {
		log.Error("Failed to set new password")
		return fmt.Errorf("password provided contains invalid characters. Must not contain backslash, space, tab, or newline")
	}
	// Every API instance gets a unique random salt. See: https://stackoverflow.com/a/1645190
	// Salt changes if the password is changed
	a.config.Salt = createRandomString(10)
	a.config.Password = hashPassword(password, a.config.Salt)

	// store hashed password in db for disaster recovery
	a.db.Set("m", "p", a.config.Password)
	a.db.Set("m", "s", a.config.Salt)
	log.Info("New password has been set")
	return nil
}

// CreateKey initialises a new key/project. This is a different concept to Redis keys.
// If the key already exists then it's name will be updated, but will otherwise be unchanged.
func (a *API) CreateKey(key string, name string) (string, error) {
	log.Debugf("Creating key %v with name %v", key, name)

	// if key is not provided then create random key of length 40
	if key == "" {
		key = createRandomString(40)
		log.Debug("Random key has been generated")
	} else if strings.ContainsAny(key, disallowedCharacters) {
		err := fmt.Errorf("key with ID '%v' is not allowed because it contains invalid characters. Keys may not contain any newlines parenthases, backslashes, etc", key)
		log.Error(err)
		return "", err
	}

	// check for disallowed ids (reserved keys)
	for _, k := range reservedKeys {
		if key == k {
			err := fmt.Errorf("key with id '%v' is not allowed because this value is reserved", key)
			return "", err
		}
	}

	// check if key already exists
	usage, exists := a.db.Get(key, "u")
	if exists {
		log.Debugf("Key already exists with usage: %v. Key name will be updated.", usage)

	} else {

		err := a.db.CreateKey(key)
		if err != nil {
			return "", err
		}

		// initialise key usage
		err = a.db.Set(key, "u", "0")
		if err != nil {
			return "", err
		}

		// initialise completed missions
		err = a.db.Set(key, "c", "")
		if err != nil {
			return "", err
		}

	}

	// set the key name (this will change the name if key already exists)
	err := a.db.Set(key, "n", name)

	if err != nil {
		return "", err
	}

	SetLoggingFile(keyLog, key)
	if exists {
		log.Infof("Updated key with ID '%s' and name '%s'", key, name)
		keyLog.Infof("Key updated. New name: '%v'", name)
	} else {
		log.Infof("Created key with ID '%s' and name '%s'", key, name)
		keyLog.Infof("Key created. Name: '%v'", name)
	}

	return key, nil
}

func (a *API) deleteKey(key string) error {
	err := a.db.DeleteKey(key)
	if err == nil {
		log.Infof("Deleted key with name '%s'", key)
	} else {
		log.Errorf("Key deletion failed: %s", err)
	}
	return err
}

// CreateMissionFromPlan creates new mission from an existing saved plan (given the name)
// or an unsaved plan (given entire plan as json). Does the following:
// - parse plan or get from db if name provided
// - validate DAG
// - assign mission ID if one is not provided
// - set start time
// - store in database
// - return created ID
func (a *API) CreateMissionFromPlan(key string, planNameOrPlan string, missionId string) (string, error) {

	var planBytes []byte

	// if plan name is provided then load plan
	if !strings.Contains(planNameOrPlan, "{") {
		if p, ok := a.db.Get(key, "p|"+planNameOrPlan); ok {
			planBytes = []byte(p)
		} else {
			keyLog.Error(&model.PlanNotFoundError{PlanName: planNameOrPlan})
			return "", &model.PlanNotFoundError{PlanName: planNameOrPlan}
		}

	} else {
		// else use the JSON provided as the temporary plan
		planBytes = []byte(planNameOrPlan)
		// note: the plan will not be saved but will still exist in the database with a '|a' (active) attribute containing
		// all the missions with this plan name. This allows these missions to still be loaded in the UI
	}

	var plan model.Plan
	err := json.Unmarshal(planBytes, &plan) // this will catch any invalid params, services, etc.
	if err != nil {
		return "", err // TODO: catch json/schema errors and give helpful response
	}

	if strings.ContainsAny(plan.Name, disallowedCharacters) {
		err := fmt.Errorf("plan with name '%v' is not allowed because it contains invalid characters", plan.Name)
		return "", err
	}

	// convert plan to mission
	m := NewMissionFromPlan(&plan)
	keyLog.Infof("Converted Plan to Mission")

	// validate graph
	validationError := m.Validate()
	if validationError != nil {
		keyLog.Errorf("Graph validation failed: %s", validationError)
		return "", validationError
	} else {
		keyLog.Infof("Validated Mission Graph")
	}

	if missionId == "" {
		// if no id is provided, use the key's usage count
		usage, _ := a.db.Get(key, "u")
		missionId = "m" + usage
		keyLog.Debugf("MissionID not provided. ID used instead: %s", missionId)

		// check if mission id already exists
		if _, exists := a.db.Get(key, missionId); exists {
			usageInt, err := strconv.Atoi(usage)
			if err != nil { // should be impossible if database uses correct schema
				usageInt = 0
			}
			loopCounter := 0
			for { // keep incrementing the number until we find a mission that doesn't already exist
				usageInt = usageInt + 1
				if usageInt > 500 {
					// this should be impossible because nobody would have this many missions at the same time
					err := fmt.Errorf("couldn't create a mission because a new mission ID could not be generated")
					log.Warnf("User couldn't create mission due to infinte loop. Key: %s", key)
					return "", err
				}
				missionId = fmt.Sprintf("m%v", usageInt)
				if _, exists := a.db.Get(key, missionId); !exists {
					break
				}
				loopCounter = loopCounter + 1
			}
		}
	} else {
		if strings.ContainsAny(missionId, disallowedCharacters) {
			err := fmt.Errorf("mission with id '%v' is not allowed because it contains invalid characters", missionId)
			return "", err
		}
		// check for disallowed ids (reserved keys)
		for _, k := range reservedKeys {
			if missionId == k {
				err := fmt.Errorf("mission with id '%v' is not allowed. Ensure that mission ID is not one of the following reserved keys: %v", missionId, strings.Join(reservedKeys, ","))
				return "", err
			}
		}
		// check if mission id already exists
		if _, exists := a.db.Get(key, missionId); exists {
			err := fmt.Errorf("mission with id '%v' already exists", missionId)
			return "", err
		}
	}
	m.Id = missionId
	m.Start = time.Now()

	// todo: this could only be a database connection error - these should be retried at least 3 times
	err = a.db.Set(key, m.Id, string(m.Bytes()))
	if err != nil {
		log.Warnf("User %s has encountered an error in CreateMissionFromPlan when updating database: %v", key, err)
		return "", err
	}

	// add to active missions - this key may not exist if plan has never been created before
	activeMissions, _ := a.db.Get(key, "a|"+m.Name)
	if activeMissions != "" {
		activeMissions += ","
	}
	activeMissions += m.Id
	err1 := a.db.Set(key, "a|"+m.Name, activeMissions)
	if err1 != nil {
		log.Warnf("User %s has encountered an error in CreateMissionFromPlan when updating database: %v", key, err1)
		return m.Id, err1 // TODO: how to recover from this err?
	}

	a.ws <- message{key, "missionCreation", m.Bytes()}

	keyLog.Infof("Mission with id '%v' has been successfully created", missionId)

	return m.Id, nil
}

// ActiveMissions finds all missions for a plan. If plan doesn't exist then an empty list is returned.
func (a *API) ActiveMissions(key string, plan string) []string {
	missions, _ := a.db.Get(key, "a|"+plan)
	if missions == "" {
		return []string{}
	}
	return strings.Split(missions, ",")
}

// AllActiveMissions finds all missions in the database for the key provided.
// Inactive/archived missions are not (or should not be) stored in the API database.
// Should only be used to check that there aren't any orphaned missions as this is less
// efficient than using ActiveMissions.
func (a *API) AllActiveMissions(key string) ([]string, error) {
	var missions []string
	allKeys, err := a.db.List(key, "")
	if err != nil {
		return missions, err
	}
	for _, s := range allKeys {
		if strings.Index(s, "|") > -1 || s == "n" || s == "u" || s == "c" {
			continue // filter out plans and reserved keys
		}
		missions = append(missions, s)
	}

	return missions, err
}

// UpdateStageState updates the state of a stage within an in-progress mission.
// POST /api/missions/[mission id]/stages/[stage name]
func (a *API) UpdateStageState(key string, missionId string, stage string, state string, ignoreDependencies bool) (mission.Response, error) {
	keyLog.Debugf("Updating stage '%s' state to '%s' in mission '%s'.", stage, state, missionId)

	var res mission.Response
	var missionBytes []byte

	// define a function to perform on a mission within a transaction
	txnFunc := func(missionString string) (string, error) {

		m, err := mission.NewFromJSON([]byte(missionString))
		if err != nil {
			// an error here is unlikely because all missions are validated before they get saved
			return "", err // TODO: catch json/schema errors and give helpful response
		}

		switch state {
		case "started":
			res, err = m.StartStage(stage, ignoreDependencies)
		case "finished":
			res, err = m.FinishStage(stage, ignoreDependencies)
		case "skipped":
			res, err = m.SkipStage(stage)
		case "failed":
			res, err = m.FailStage(stage)
		case "excluded", "ignored":
			res, err = m.ExcludeStage(stage)
		default:
			err = fmt.Errorf("invalid stage state '%v'; choose one of started, finished, failed, skipped, or excluded", state)
		}

		if err != nil {
			keyLog.Errorf("Error when updating stage %s's state to %s in mission %s: %s", stage, state, missionId, err)
			return "", err
		} else {
			keyLog.Infof("Stage %s in mission %s has been set to %v", stage, missionId, state)
		}

		missionBytes = m.Bytes()

		return string(missionBytes), err
	}

	// The mission is operated on within a transaction to prevent any other updates while this update is in progress.
	// Retry the transaction at least 3 times: it is highly likely that the key will be unlocked within milliseconds,
	// but any more than 10 risks creating issues for the client as the request will take a minimum of 140ms
	// If the key is still locked after 3 attempts then the API will return 429, which causes the client to retry.
	var err error
	for attempts := 0; attempts < 3; attempts++ {
		err = a.db.DoTransaction(txnFunc, key, missionId)
		if err != nil {
			switch err.(type) {
			case *model.TransactionFailedError:
				keyLog.Debugf("Got 'TransactionFailedError' when ending the stage. This is attempt number %v.\n", attempts+1)
				time.Sleep(10 * time.Millisecond * time.Duration((attempts+1)^2))
				// retry the transaction
			default:
				break // error will be returned
			}
		} else {
			break
		}
	}

	// if update was successful then send the updated mission to all websocket clients
	if err == nil {
		a.ws <- message{key, "missionUpdate", missionBytes}
	}

	// if the mission is complete, add it to the list of missions to be cleaned up
	if res.IsComplete {
		a.ws <- message{key, "missionCompleted", missionBytes}
		completedList := append(a.CompletedMissions(key), missionId)
		completedListBytes := strings.Join(completedList, ",")
		a.db.Set(key, "c", completedListBytes)
		keyLog.Infof("Mission %s is complete", missionId)
	}

	return res, err
}

// CompletedMissions returns a list all missionIds that are completed so that they can be archived and deleted.
func (a *API) CompletedMissions(key string) []string {
	completedListString, ok := a.db.Get(key, "c")
	if !ok || completedListString == "" { // this should never happen
		log.Warnf("Completed mission string is empty for key '%s'.", key)
		return []string{}
	}
	completedList := strings.Split(completedListString, ",")
	return completedList
}

// SavePlan stores a new plan in the database if that plan is valid. Current behaviour is to overwrite existing plans.
func (a *API) SavePlan(key string, plan model.Plan) error {

	// convert plan to mission for validation of graph only
	m := NewMissionFromPlan(&plan)
	err := m.Validate()
	if err != nil {
		return err
	}

	planBytes, _ := json.Marshal(plan)
	keyLog.Infof("Converted Plan '%s' to Mission", plan.Name)
	p, _ := a.db.Get(key, "p|"+plan.Name)
	err = a.db.Set(key, "p|"+plan.Name, string(planBytes))
	// if plan already exists, do not re-create the 'active' key
	if p == "" {
		err = a.db.Set(key, "a|"+plan.Name, "")
	}
	if err != nil {
		keyLog.Errorf("Error when saving plan to database: %v", err)
		log.Warnf("User %s encountered error when saving plan to database: %v", key, err)
		return err
	}
	keyLog.Infof("Plan '%s' has been saved.", plan.Name)
	a.ws <- message{key, "planCreation", planBytes}
	return nil
}

// ListPlans returns all plan names.
// The complete list of plans is the union of all saved plans and all active plans
func (a *API) ListPlans(key string) ([]string, error) {

	plans, err := a.db.List(key, "p")
	for i, s := range plans {
		plans[i] = strings.Replace(s, "p|", "", 1)
	}

	keyLog.Debugf("Found %v saved plans", len(plans))

	activePlans, err := a.db.List(key, "a|")
	keyLog.Debugf("Found %v active plans", len(activePlans))

Loop:
	for _, s := range activePlans {
		s = strings.Replace(s, "a|", "", 1)

		for _, s2 := range plans {
			if s == s2 {
				continue Loop
			}
		}
		keyLog.Debugf("Found active and unsaved plan '%v'", s)
		plans = append(plans, s)
	}
	keyLog.Debugf("Found a total of %v plans", len(plans))
	return plans, err
}

// initDashboard starts serving the mission dashboard web app.
// This should not run if config.Dashboard.Enabled is set to false.
func (a *API) initDashboard() {
	var html []byte
	var err error
	if a.config.Dashboard.Enabled {
		// serve the houston console
		a.router.HandleFunc("/console", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<html><link rel="stylesheet" type="text/css" href="https://storage.googleapis.com/houston-static/console/main.css"><script src="https://storage.googleapis.com/houston-static/console/main.js"></script></html>`))
		})
		log.Infof("Initialised console UI")

		if a.config.Dashboard.Src == "" {
			html = []byte(`<!doctype html><html lang="en"><head><meta charset="utf-8"/>
 <link rel="icon" href="https://callhouston.io/houston-favicon.png"/>
 <meta name="viewport" content="width=device-width,initial-scale=1"/>
 <meta name="theme-color" content="#000000"/>
 <meta name="description" content="Houston Dashboard"/>
 <script defer="defer" src="https://storage.googleapis.com/houston-static/dashboard/main.js"></script>
 <link href="https://storage.googleapis.com/houston-static/dashboard/main.css" rel="stylesheet">
</head><body><noscript>You need to enable JavaScript to run this app.</noscript><div id="root"></div></body></html>
`)
			log.Infof("Using default dashboard UI")

		} else {
			html, err = os.ReadFile(a.config.Dashboard.Src)
			if err != nil {
				log.Error("Couldn't load custom dashboard UI HTML")
				log.Error(err)
				panic(err)
			}
			log.Infof("Successfully loaded Custom dashboard UI HTML from %v", a.config.Dashboard.Src)
		}

		a.router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write(html)
		})

		var url string
		if a.protocol == "https" {
			url = fmt.Sprintf("https://%v", a.config.TLS.Host)
		} else {
			url = fmt.Sprintf("http://localhost:%v", a.config.Port)
		}
		msg := "Mission dashboard is live on " + url
		log.Info(msg)
		if isTerminal {
			fmt.Println("🔭 " + msg)
		}
	}
}

// Run starts the API server
func (a *API) Run() {

	var err error
	if a.protocol == "https" {

		msg := fmt.Sprintf("Houston ready to receive calls on https://%v/api/v1", a.config.TLS.Host)
		log.Info(msg)
		if isTerminal {
			fmt.Println("📡 " + msg)
		}

		if a.config.TLS.Auto {
			// use the ACME protocol to generate and renew certificates automatically
			log.Infof("Automatic TLS is enabled. Houston will attempt to generate a certificate for %s.\n", a.config.TLS.Host)

			certManager := autocert.Manager{
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(a.config.TLS.Host),
				Cache:      autocert.DirCache("certs"),
			}

			server := &http.Server{
				Addr:    ":https",
				Handler: a.router,
				TLSConfig: &tls.Config{
					GetCertificate: certManager.GetCertificate,
					MinVersion:     tls.VersionTLS12,
				},
			}

			go http.ListenAndServe(":http", certManager.HTTPHandler(nil))

			err = server.ListenAndServeTLS("", "") // key and cert are coming from Let's Encrypt

		} else {
			// if self-managed certificates are provided
			err = http.ListenAndServeTLS(":https", a.config.TLS.CertFile, a.config.TLS.KeyFile, a.router)
		}
	} else {
		msg := fmt.Sprintf("Houston ready to receive calls on http://localhost:%v/api/v1", a.config.Port)
		log.Info(msg)
		if isTerminal {
			fmt.Println("📡 " + msg)
		}

		err = http.ListenAndServe(":"+a.config.Port, a.router)
	}

	if err != nil {
		log.Error(err)
		panic(err)
	}

}

func init() {
	rand.Seed(time.Now().UnixNano()) // change random seed
}

func main() {

	if err := func() (rootCmd *cobra.Command) {

		rootCmd = &cobra.Command{
			Use:   "houston",
			Short: "HOUSTON Orchestration API · https://callhouston.io",
			Args:  cobra.ArbitraryArgs,
			Run: func(c *cobra.Command, args []string) {
				s := "\u001B[1;38;2;58;145;172m"
				e := "\u001B[0m"
				fmt.Println("\n🚀 \u001B[1mHOUSTON\u001B[0m · Orchestration API · https://callhouston.io\nBasic usage:")
				fmt.Printf("  %[1]vhouston api%[2]v                    \u001B[37m# starts a local API server%[2]v\n", s, e)
				fmt.Printf("  %[1]vhouston save%[2]v \u001B[1m--plan plan.yaml%[2]v  \u001B[37m# saves a new plan%[2]v\n", s, e)
				fmt.Printf("  %[1]vhouston start%[2]v \u001B[1m--plan my-plan%[2]v   \u001B[37m# creates and triggers a new mission%[2]v\n", s, e)
				fmt.Printf("  %[1]vhouston help%[2]v                   \u001B[37m# shows help for all commands%[2]v\n", s, e)
				return
			},
		}

		rootCmd.AddCommand(func() (createCmd *cobra.Command) {
			createCmd = &cobra.Command{
				Use:   "version",
				Short: "Print the version number",
				Run: func(c *cobra.Command, args []string) {
					fmt.Println("v0.4.1")
				},
			}
			return
		}())

		rootCmd.AddCommand(func() (createCmd *cobra.Command) {
			createCmd = &cobra.Command{
				Use:   "api",
				Short: "Run the Houston API server",
				Run: func(c *cobra.Command, args []string) {
					configPath, _ := createCmd.Flags().GetString("config")
					api := New(configPath)
					api.Run()
				},
			}
			createCmd.Flags().String("config", "", "path to a config file")
			return
		}())

		rootCmd.AddCommand(func() (createCmd *cobra.Command) {
			var id = ""
			var name = ""
			var password = ""
			createCmd = &cobra.Command{
				Use:   "create-key",
				Short: "Create a new API key. Requires admin password.",
				Run: func(c *cobra.Command, args []string) {
					err := client.CreateKey(id, name, password)
					if err != nil {
						client.HandleCommandLineError(err)
					}
				},
			}
			createCmd.Flags().StringVarP(&id, "id", "i", "", "New API key value")
			createCmd.Flags().StringVarP(&name, "name", "n", "", "Description for this key")
			createCmd.Flags().StringVarP(&password, "password", "p", "", "API admin password")

			return
		}())

		rootCmd.AddCommand(func() (createCmd *cobra.Command) {
			var plan string
			createCmd = &cobra.Command{
				Use:   "save",
				Short: "Save a plan",
				Run: func(c *cobra.Command, args []string) {
					err := client.Save(plan)
					if err != nil {
						client.HandleCommandLineError(err)
					}
				},
			}
			createCmd.Flags().StringVarP(&plan, "plan", "p", "", "File path or URL of the plan to save.")
			createCmd.MarkFlagRequired("plan")
			return
		}())

		rootCmd.AddCommand(func() (createCmd *cobra.Command) {
			var plan string
			var missionId = ""
			var stages = ""
			var exclude = ""
			var skip = ""
			createCmd = &cobra.Command{
				Use:   "start",
				Short: "Create a new mission and trigger the first stage(s)",
				Run: func(c *cobra.Command, args []string) {
					err := client.Start(plan, missionId,
						strings.Split(strings.Replace(stages, " ", "", -1), ","),
						strings.Split(strings.Replace(exclude, " ", "", -1), ","),
						strings.Split(strings.Replace(skip, " ", "", -1), ","))
					if err != nil {
						client.HandleCommandLineError(err)
					}
				},
			}
			createCmd.Flags().StringVarP(&plan, "plan", "p", "", "Name or file path of the plan to create a new mission with")
			createCmd.MarkFlagRequired("plan")
			createCmd.Flags().StringVarP(&missionId, "mission-id", "m", "", "Mission ID to assign to the new mission")
			createCmd.Flags().StringVarP(&stages, "stages", "s", "", "Comma separated list of stage names to be used as the starting point for the mission. \nIf not provided, all stages with no upstream stages will be triggered")
			createCmd.Flags().StringVarP(&exclude, "exclude", "i", "", "Comma separated list of stage names to be excluded in the new mission")
			createCmd.Flags().StringVarP(&skip, "skip", "k", "", "Comma separated list of stage names to be skipped in the new mission")
			return
		}())

		rootCmd.AddCommand(func() (createCmd *cobra.Command) {
			createCmd = &cobra.Command{
				Use:   "demo",
				Short: "Run the API in demo mode",
				Run: func(c *cobra.Command, args []string) {
					demo(createCmd)
				},
			}
			createCmd.Flags().String("config", "", "path to a config file")
			return
		}())

		return
	}().Execute(); err != nil {
		panic(err)
	}
}
