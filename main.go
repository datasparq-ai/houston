package main

import (
	"encoding/json"
	"fmt"
	"github.com/datasparq-ai/houston/client"
	"github.com/datasparq-ai/houston/database"
	"github.com/datasparq-ai/houston/mission"
	"github.com/datasparq-ai/houston/model"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type API struct {
	db     database.Database
	router *mux.Router
	ws     chan message
	config *Config
}

// @title Houston API
// @version 1.0
// @description Houston API documentation. You can visit the GitHub repository at https://github.com/datasparq-ai/houston

// @contact.name Datasparq
// @contact.url
// @contact.email

// @license.name
// @license.url

// @host localhost:8080
// @BasePath /api/v1

// New creates the Houston API object.
// It will create or connect to a database depending on the settings in the config file.
// local db will only persist while program is running.
func New(configPath string) API {

	config := LoadConfig(configPath)

	var db database.Database
	// attempt to connect to redis - if not found then use local db
	db = database.NewRedisDatabase(config.Redis.Addr, config.Redis.Password, config.Redis.DB)
	err := db.Ping()
	switch e := err.(type) {
	case nil:
		fmt.Printf("🚨 Connected to Redis Database at %v\n", config.Redis.Addr)
	case *net.OpError:
		switch e.Err.(type) {
		case *os.SyscallError:
			// TODO: fail in production mode (and unittest mode)
			fmt.Printf("Couldn't connect to Redis Database at %v. Using in-memory database.\n", config.Redis.Addr)
			db = database.NewLocalDatabase()
		case *net.AddrError:
			panic(err) // this happens when user puts protocol in Redis.Addr
		default:
			panic(err)
		}
	default:
		panic(err)
	}

	a := API{db, nil, nil, config}

	config.Password = strings.Trim(config.Password, " \n\t")
	if config.Password != "" {
		err := a.SetPassword(config.Password)
		if err != nil {
			log.Fatal(err)
		}
	}

	a.initRouter()
	a.initDashboard()
	a.initWebSocket()
	return a
}

func (a *API) SetPassword(password string) error {
	if len(password) < 10 {
		return fmt.Errorf("Password provided is not long enough. Houston admin password must be at least 10 characters. Recommended length is 30.")
	}
	if strings.ContainsAny(password, "\\ \t\n") {
		return fmt.Errorf("Password provided contains invalid characters. Must not contain backslash, space, tab, or newline.")
	}
	// Every API instance gets a unique random salt. See: https://stackoverflow.com/a/1645190
	// Salt changes if the password is changed
	a.config.Salt = createRandomString(10)
	a.config.Password = hashPassword(password, a.config.Salt)
	return nil
}

// CreateKey initialises a new key/project. This is a different concept to Redis keys.
func (a *API) CreateKey(key string, name string) (string, error) {

	// if key is not provided then create random key of length 40
	if key == "" {
		key = createRandomString(40)
	}

	err := a.db.CreateKey(key)
	if err != nil {
		return "", err
	}

	err1 := a.db.Set(key, "n", name)
	err2 := a.db.Set(key, "u", "0") // usage
	err3 := a.db.Set(key, "c", "")  // completed missions

	if err1 != nil || err2 != nil || err3 != nil {
		return "", err1
	}

	return key, nil
}

func (a *API) DeleteKey(key string) error {
	err := a.db.DeleteKey(key)
	return err
}

// CreateMissionFromPlan creates new mission from an existing saved plan (given the name)
//
//	or an unsaved plan (given entire plan as json). Does the following:
//
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
			return "", fmt.Errorf("no plan found named '%v'", planNameOrPlan)
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

	if strings.ContainsAny(plan.Name, string(disallowedCharacters)) {
		return "", fmt.Errorf("plan with name '%v' is not allowed because it contains invalid characters", plan.Name)
	}

	// convert plan to mission
	m := NewMissionFromPlan(&plan)

	// validate graph
	validationError := m.Validate()
	if validationError != nil {
		return "", validationError
	}

	if missionId == "" {
		// if no id is provided, use the key's usage count
		usage, _ := a.db.Get(key, "u")
		missionId = "m" + usage

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
					return "", fmt.Errorf("couldn't create a mission because a new mission ID could not be generated")
				}
				missionId = fmt.Sprintf("m%v", usageInt)
				if _, exists := a.db.Get(key, missionId); !exists {
					break
				}
				loopCounter = loopCounter + 1
			}
		}
	} else {
		if strings.ContainsAny(missionId, string(disallowedCharacters)) {
			return "", fmt.Errorf("mission with id '%v' is not allowed because it contains invalid characters", missionId)
		}
		// check for disallowed ids (reserved keys)
		for _, k := range reservedKeys {
			if missionId == k {
				return "", fmt.Errorf("mission with id '%v' is not allowed", missionId)
			}
		}
		// check if mission id already exists
		if _, exists := a.db.Get(key, missionId); exists {
			return "", fmt.Errorf("mission with id '%v' already exists", missionId)
		}
	}
	m.Id = missionId
	m.Start = time.Now()

	a.db.Set(key, m.Id, string(m.Bytes()))

	// add to active missions - this key may not exist if plan has never been created before
	activeMissions, _ := a.db.Get(key, "a|"+m.Name)
	if activeMissions != "" {
		activeMissions += ","
	}
	activeMissions += m.Id
	err1 := a.db.Set(key, "a|"+m.Name, activeMissions)
	if err1 != nil {
		return m.Id, err1 // TODO: how to recover from this err?
	}

	a.ws <- message{key, "missionCreation", m.Bytes()}

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
			return "", err
		}

		missionBytes = m.Bytes()

		return string(missionBytes), err
	}

	// The mission is operated on within a transaction to prevent any other updates while this update is in progress.
	// If the mission is currently locked then the API will return a 429 error, which causes the client to retry.
	err := a.db.DoTransaction(txnFunc, key, missionId)

	//if err != missionNotFound? {
	//  return res, fmt.Errorf("mission with id '%v' not found", missionId)
	//}

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
	}

	return res, err
}

// CompletedMissions returns a list all missionIds that are completed so that they can be archived and deleted.
func (a *API) CompletedMissions(key string) []string {
	completedListString, ok := a.db.Get(key, "c")
	if !ok || completedListString == "" { // this should never happen
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

	p, _ := a.db.Get(key, "p|"+plan.Name)
	err = a.db.Set(key, "p|"+plan.Name, string(planBytes))
	// if plan already exists, do not re-create the 'active' key
	if p == "" {
		err = a.db.Set(key, "a|"+plan.Name, "")
	}
	if err != nil {
		return err
	}
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

	activePlans, err := a.db.List(key, "a|")
Loop:
	for _, s := range activePlans {
		s = strings.Replace(s, "a|", "", 1)

		for _, s2 := range plans {
			if s == s2 {
				continue Loop
			}
		}
		plans = append(plans, s)
	}
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
			w.Write([]byte(`<html><link rel="stylesheet" type="text/css" href="https://game-ii.s3.eu-west-2.amazonaws.com/console/main.css"><script src="https://game-ii.s3.eu-west-2.amazonaws.com/console/main.js"></script></html>`))
		})
		if a.config.Dashboard.Src == "" {
			// TODO: host on static.callhouston.io
			html = []byte(`<!doctype html><html lang="en"><head><meta charset="utf-8"/>
 <link rel="icon" href="https://callhouston.io/houston-favicon.png"/>
 <meta name="viewport" content="width=device-width,initial-scale=1"/>
 <meta name="theme-color" content="#000000"/>
 <meta name="description" content="Houston Dashboard"/>
 <script defer="defer" src="https://game-ii.s3.eu-west-2.amazonaws.com/dashboard/main.js"></script>
 <link href="https://game-ii.s3.eu-west-2.amazonaws.com/dashboard/main.css" rel="stylesheet">
</head><body><noscript>You need to enable JavaScript to run this app.</noscript><div id="root"></div></body></html>
`)

		} else {
			html, err = os.ReadFile(a.config.Dashboard.Src)
			if err != nil {
				panic(err)
			}
		}

		a.router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write(html)
		})
	}

	fmt.Printf("🔭 Mission dashboard is live on http://localhost:%v\n", a.config.Port)
}

func (a *API) Run() {
	fmt.Printf("📡 Houston ready to receive calls on http://localhost:%v/api/v1\n", a.config.Port)
	err := http.ListenAndServe(":"+a.config.Port, a.router)
	if err != nil {
		panic(err)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano()) // change random seed

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
					fmt.Println("v0.1.1")
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
						panic(err)
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
						panic(err)
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
						panic(err)
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
		log.Panicln(err)
	}
}
