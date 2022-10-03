package client

import (
  "bytes"
  "encoding/json"
  "fmt"
  "github.com/datasparq-ai/houston/model"
  "io/ioutil"
  "net/http"
  "time"
)

func (client *Client) request(method, path string, body []byte) *http.Response {
  url := client.BaseUrl + path
  req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
  if err != nil {
    fmt.Println("Got an error creating request to url: " + url)
    // TODO: return error
    panic(err)
  }
  req.Header.Set("x-access-key", client.Key)
  req.Header.Set("Content-Type", "application/json")

  if client.Auth.Password != "" {
    req.SetBasicAuth(client.Auth.Username, client.Auth.Password)
  }

  httpClient := &http.Client{}
  resp, err := httpClient.Do(req)
  if err != nil {
    panic(err)
  }
  //fmt.Println(resp.Status)
  if resp.StatusCode == http.StatusTooManyRequests {
    // wait and retry up to 100 times
    loopCounter := 0
    for resp.StatusCode == http.StatusTooManyRequests && loopCounter < 100 {
      time.Sleep(time.Millisecond * 100)
      loopCounter++

      // recreate request
      req, _ = http.NewRequest(method, url, bytes.NewBuffer(body))
      req.Header.Set("x-access-key", client.Key)

      req.Header.Set("Content-Type", "application/json")
      resp, err = httpClient.Do(req)
      if err != nil {
        fmt.Println("Got an error when requesting url: " + url)
        // TODO: return error
        panic(err)
      }
    }
  }
  return resp
}
func (client *Client) post(path string, body []byte) *http.Response {
  return client.request("POST", path, body)
}
func (client *Client) get(path string) *http.Response {
  return client.request("GET", path, []byte{})
}
func (client *Client) delete(path string) *http.Response {
  return client.request("DELETE", path, []byte{})
}

func (client *Client) postKey(reqBody model.Key) (string, error) {
  reqJSON, _ := json.Marshal(reqBody)
  resp := client.post("/key", reqJSON)
  responseBody, err := ioutil.ReadAll(resp.Body)
  if resp.StatusCode != http.StatusOK {
    if err != nil {
      return "", handleInvalidResponse(err)
    }
    return "", handleErrorResponse(responseBody)
  }
  return string(responseBody), nil
}

func (client *Client) postMissions(reqBody model.MissionCreateRequest) (model.MissionCreatedResponse, error) {
  var missionResponse model.MissionCreatedResponse
  reqJSON, _ := json.Marshal(reqBody)
  resp := client.post("/missions", reqJSON)

  err := parseResponse(resp, &missionResponse)
  return missionResponse, err
}

func (client *Client) postMissionsStages(mission, stage string, reqBody model.MissionStageStateUpdate) (model.MissionStageStateUpdateResponse, error) {
  var missionResponse model.MissionStageStateUpdateResponse
  path := fmt.Sprintf("/missions/%v/stages/%v", mission, stage)
  reqJSON, _ := json.Marshal(reqBody)
  resp := client.post(path, reqJSON)
  err := parseResponse(resp, &missionResponse)
  return missionResponse, err
}

func (client *Client) postPlans(reqBody []byte) error {
  resp := client.post("/plans", reqBody)
  responseBody, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    return handleInvalidResponse(err)
  }
  if resp.StatusCode != 200 {
    return handleErrorResponse(responseBody)
  }
  return nil
}
