package client

import (
	"encoding/json"
	"fmt"
	"github.com/datasparq-ai/houston/model"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
	"strings"
)

type Auth struct {
	Username string
	Password string
}

type Client struct {
	BaseUrl string
	Key     string
	Auth    Auth
}

// New creates a Houston client instance. One instance uses one API key to make all requests.
func New(key string, baseUrl string) Client {

	if key == "" {
		key = os.Getenv("HOUSTON_KEY")
		// if key is a valid URI that contains the baseUrl and the key
		if strings.HasPrefix(key, "http://") || strings.HasPrefix(key, "https://") {
			splitKey := strings.Split(key, "/key/")
			if len(splitKey) != 2 {
				fmt.Printf("Key has an invalid format. Expected format: '{base URL}/key/{key ID}'.\n")
				os.Exit(1)
			}
			baseUrl = splitKey[0]
			key = splitKey[1]
		}
	}

	if baseUrl == "" {
		baseUrl = os.Getenv("HOUSTON_BASE_URL")
	}
	if baseUrl == "" {
		// look for locally running houston server before defaulting to callhouston.io
		baseUrl = "http://localhost:8000/api/v1"
		req, _ := http.NewRequest("GET", baseUrl, nil)
		httpClient := &http.Client{}
		_, err := httpClient.Do(req)
		if err != nil {
			fmt.Println("A base URL (e.g. 'http://localhost:8000/api/v1') was not provided, and no Houston API was not found locally. Provide the base URL with the 'HOUSTON_BASE_URL' environment variable.")
			baseUrl = "https://callhouston.io/api/v1"
		}
	} else if !(strings.HasPrefix(baseUrl, "http://") || strings.HasPrefix(baseUrl, "https://")) {
		fmt.Printf("Base URL '%s' isn't a valid URL; must start with either 'http://' or 'https://'\n", baseUrl)
		os.Exit(1)
	} else {
		baseUrl = strings.TrimSuffix(baseUrl, "/")
	}

	// automatically use admin credentials stored in environment
	var auth Auth
	envPass := os.Getenv("HOUSTON_PASSWORD")
	if envPass != "" {
		auth = Auth{"admin", envPass}
	}

	client := Client{baseUrl, key, auth}

	// Check the health of the selected API server. This will only produce a warning if it fails.
	healthCheckError := healthCheck(baseUrl)
	if healthCheckError != nil {
		fmt.Printf("Warning: Server health check failed. Check that the URL '%v' is correct and the server is running. Error: %v\n", baseUrl, healthCheckError.Error())
		if !strings.HasSuffix(baseUrl, "/api/v1") {
			fmt.Println("Warning: Base URL doesn't end with '/api/v1', which is the standard base path.")
		}
	}

	return client
}

func (client *Client) GetMission(missionId string) (model.Mission, error) {
	var mission model.Mission
	resp := client.get("/missions/" + missionId)
	err := parseResponse(resp, &mission)
	return mission, err
}

func loadPlan(plan string) (string, error) {
	// TODO: load from gs:// or s3:// or http://
	// if file path is provided then load file
	if strings.HasSuffix(plan, ".json") {
		data, err := os.ReadFile(plan)
		if err != nil {
			return "", err
		}
		plan = string(data)
	} else if strings.HasSuffix(plan, ".yaml") {
		// if yaml then convert to API representation (JSON)
		data, err := os.ReadFile(plan)
		if err != nil {
			return "", err
		}
		var p map[string]interface{}
		err = yaml.Unmarshal(data, &p)
		if err != nil {
			return "", err
		}
		planBytes, err := json.Marshal(p)
		if err != nil {
			return "", err
		}
		plan = string(planBytes)
	}
	return plan, nil
}

// CreateMission will create a new mission from a plan or plan name. If a plan name is provided then it must correspond
// to a plan saved with SavePlan.
func (client *Client) CreateMission(plan string, id string) (model.MissionCreatedResponse, error) {
	plan, err := loadPlan(plan) // if file path was provided then load file, else do nothing
	if err != nil {
		panic(err)
	}
	res, err := client.postMissions(model.MissionCreateRequest{Plan: plan, Id: id})
	return res, err
}

func (client *Client) ListActiveMissions() ([]string, error) {
	var missions []string
	resp := client.get("/missions")
	err := parseResponse(resp, &missions)
	return missions, err
}

func (client *Client) DeleteMission(missionId string) (model.Mission, error) {
	mission, err := client.GetMission(missionId)
	if err != nil {
		return mission, err
	}
	var deleteResponse model.Success
	res := client.delete("/missions/" + missionId)
	err = parseResponse(res, &deleteResponse)
	return mission, err
}

func (client *Client) StartStage(mission, stage string, ignoreDependencies bool) (model.MissionStageStateUpdateResponse, error) {
	reqBody := model.MissionStageStateUpdate{State: "started", IgnoreDependencies: ignoreDependencies}
	return client.postMissionsStages(mission, stage, reqBody)
}
func (client *Client) FinishStage(mission, stage string, ignoreDependencies bool) (model.MissionStageStateUpdateResponse, error) {
	reqBody := model.MissionStageStateUpdate{State: "finished", IgnoreDependencies: ignoreDependencies}
	return client.postMissionsStages(mission, stage, reqBody)
}
func (client *Client) FailStage(mission, stage string) (model.MissionStageStateUpdateResponse, error) {
	reqBody := model.MissionStageStateUpdate{State: "failed", IgnoreDependencies: false}
	return client.postMissionsStages(mission, stage, reqBody)
}
func (client *Client) ExcludeStage(mission, stage string) (model.MissionStageStateUpdateResponse, error) {
	reqBody := model.MissionStageStateUpdate{State: "excluded"}
	return client.postMissionsStages(mission, stage, reqBody)
}
func (client *Client) SkipStage(mission, stage string) (model.MissionStageStateUpdateResponse, error) {
	reqBody := model.MissionStageStateUpdate{State: "skipped"}
	return client.postMissionsStages(mission, stage, reqBody)
}

func (client *Client) SavePlan(filePath string) error {
	plan, err := loadPlan(filePath)
	if err != nil {
		panic(err)
	}
	return client.postPlans([]byte(plan))
}

func (client *Client) CreateKey(id, name, password string) (string, error) {
	key := model.Key{
		Id:   id,
		Name: name,
	}
	if password != "" {
		client.Auth.Username = "admin"
		client.Auth.Password = password
	}
	createdKey, err := client.postKey(key)
	return createdKey, err
}

func (client *Client) GetPlan(name string) (model.Plan, error) {
	var plan model.Plan
	resp := client.get("/plans/" + name)
	err := parseResponse(resp, &plan)
	return plan, err
}

func (client *Client) DeletePlan(name string) error {
	var success model.Success
	resp := client.delete("/plans/" + name)
	err := parseResponse(resp, &success)
	return err
}

func (client *Client) ListPlans() ([]string, error) {
	var plans []string
	resp := client.get("/plans")
	err := parseResponse(resp, &plans)
	return plans, err
}

func (client *Client) ListKeys(password string) ([]string, error) {
	var keys []string
	if password != "" {
		client.Auth.Username = "admin"
		client.Auth.Password = password
	}
	resp := client.get("/key/all")
	err := parseResponse(resp, &keys)
	return keys, err
}

func (client *Client) DeleteKey(password string) error {
	var success model.Success
	if password != "" {
		client.Auth.Username = "admin"
		client.Auth.Password = password
	}
	resp := client.delete("/key")
	err := parseResponse(resp, &success)
	return err
}
