package client

import (
	"encoding/json"
	"fmt"
	"github.com/datasparq-ai/houston/model"
	"gopkg.in/yaml.v3"
	"io"
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
			fmt.Println("Defaulting to callhouston.io")
			baseUrl = "https://callhouston.io/api/v1"
		}
	} else {
		baseUrl = strings.TrimSuffix(baseUrl, "/")
		// todo: verify BaseURL: must start with http
	}

	// automatically use admin credentials stored in environment
	var auth Auth
	envPass := os.Getenv("HOUSTON_PASSWORD")
	if envPass != "" {
		auth = Auth{"admin", envPass}
	}

	return Client{baseUrl, key, auth}
}

func (client *Client) GetMission(missionId string) (model.Mission, error) {
	resp := client.get("/missions/" + missionId)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.Mission{}, err // TODO: format error into helpful message
	}
	if resp.StatusCode > 200 {
		var errorResponse model.Error
		err = json.Unmarshal(responseBody, &errorResponse)
		if err != nil {
			return model.Mission{}, handleInvalidResponse(err)
		}
		err := fmt.Errorf(errorResponse.Message)
		return model.Mission{}, err
	}

	var missionResponse model.Mission
	err = json.Unmarshal(responseBody, &missionResponse)
	if err != nil {
		return model.Mission{}, handleInvalidResponse(err)

	}
	return missionResponse, nil
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
	resp := client.get("/missions")
	responseBody, err := io.ReadAll(resp.Body)
	var missions []string
	if err != nil {
		return missions, handleInvalidResponse(err)
	}
	if resp.StatusCode != 200 {
		var errorResponse model.Error
		err = json.Unmarshal(responseBody, &errorResponse)
		if err != nil {
			return missions, handleInvalidResponse(err)
		}
		err := fmt.Errorf(errorResponse.Message)
		return missions, err
	}
	err = json.Unmarshal(responseBody, &missions)
	if err != nil {
		return missions, handleInvalidResponse(err)
	}
	return missions, nil
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
	resp := client.delete("/plans/" + name)
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return handleInvalidResponse(err)
	}
	if resp.StatusCode != 200 {
		return handleErrorResponse(responseBody)
	}
	return nil
}

func (client *Client) ListPlans() ([]string, error) {
	var plans []string
	resp := client.get("/plans")
	err := parseResponse(resp, &plans)
	return plans, err
}

func (client *Client) ListKeys() ([]string, error) {
	var keys []string
	resp := client.get("/key/all")
	err := parseResponse(resp, &keys)
	return keys, err
}

func (client *Client) DeleteKey() (error) {
	resp := client.delete("/key")
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return handleInvalidResponse(err)
	}
	if resp.StatusCode != 200 {
		return handleErrorResponse(responseBody)
	}
	return nil
}

