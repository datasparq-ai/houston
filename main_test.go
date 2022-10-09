/**
Tests for the API. Runs a local Houston API server in a goroutine and uses Houston's Go client to run tests.
One key 'test' is used for all tests. Each test creates a new mission, with the ID set to the name of the test.
*/

package main

import (
  "encoding/json"
  "fmt"
  "github.com/datasparq-ai/houston/client"
  "github.com/datasparq-ai/houston/model"
  "io/ioutil"
  "os"
  "testing"
)

func TestMain(m *testing.M) {

  api := New("")
  api.DeleteKey("test")
  api.CreateKey("test", "unittest-1")
  go api.Run()

  code := m.Run() // run tests

  api.DeleteKey("test") // clean up
  os.Exit(code)
}

func TestAPI_CreateKey(t *testing.T) {

  api := New("")
  key, err := api.CreateKey("", "") // generate random key
  if err != nil {
    t.Fatalf(`Could not create key`)
  }
  if key == "" {
    t.Fatalf(`Random key not created; key is empty string`)
  }

  key2, _ := api.CreateKey("", "")
  if key == key2 {
    t.Fatalf(`Key is not random`)
  }

  err = api.DeleteKey(key) // clean up
  if err != nil {
    t.Fatalf(`Could not delete key`)
  }
}

func TestAPI_GetMission(t *testing.T) {

  c := client.New("test", "")

  data, _ := ioutil.ReadFile("tests/test_plan.json")
  var originalPlan model.Plan
  _ = json.Unmarshal(data, &originalPlan)

  res, err := c.CreateMission(string(data), "TestAPI_GetMission")
  if err != nil {
    t.Fatalf(`Could not create mission`)
  }
  missionId := res.Id

  m, err := c.GetMission(missionId)
  if err != nil {
    t.Fatalf(`Could not get mission`)
  }
  if m.Name != originalPlan.Name {
    t.Fatalf("Got the wrong mission/plan back!")
  }

  missions, err := c.ListActiveMissions()
  missionExists := false
  for i := range missions {
    if missions[i] == missionId {
      missionExists = true
    }
  }
  if !missionExists {
    t.Fatalf("Mission not found when listing missions")
  }
}

func TestAPI_PostMissionStage(t *testing.T) {

  c := client.New("test", "")

  data, _ := ioutil.ReadFile("tests/test_plan.json")
  res, err := c.CreateMission(string(data), "TestAPI_PostMissionStage")
  if err != nil {
    t.Fatalf(`Could not create mission`)
  }
  missionId := res.Id

  activeMissions, err := c.ListActiveMissions()
  if err != nil {
    t.Fatalf("Got an error trying to list active missions")
  }
  if len(activeMissions) < 1 {
    t.Fatalf("There should be at least one active mission")
  }
  isThisMissionListed := false
  for _, m := range activeMissions {
    if m == "n" {
      t.Fatalf("Reserved key was listed as a mission")
    }
    if m != missionId {
      isThisMissionListed = true
      break
    }
  }
  if !isThisMissionListed {
    t.Fatalf("The active mission list should contain the same ID as the one we just created")
  }

  r, err := c.StartStage(missionId, "stage-1", false)
  if err != nil || !r.Success {
    t.Fatalf(`Could not start stage`)
  }

  _, err = c.StartStage(missionId, "stage-1", false)
  if err == nil {
    t.Fatalf(`Didn't get an error when starting stage twice`)
  }

  _, err = c.FinishStage(missionId, "stage-1", false)
  if err != nil {
    t.Fatalf(`Could not finish stage`)
  }
}

func TestAPI_PostMissionStage_StartStage_IgnoreDependencies(t *testing.T) {

  c := client.New("test", "")

  data, _ := ioutil.ReadFile("tests/test_plan.json")
  res, err := c.CreateMission(string(data), "TestAPI_PostMissionStage_StartStage_IgnoreDependencies")
  if err != nil {
    t.Fatalf(`Could not create mission`)
  }
  missionId := res.Id

  r, err := c.StartStage(missionId, "stage-2", true)
  if err != nil || !r.Success {
    t.Fatalf(`Could not start stage despite ignoring dependencies`)
  }

  _, err = c.StartStage(missionId, "stage-1", false)
  if err == nil {
    t.Fatalf(`Didn't get an error when starting excluded stage`)
  }

  r, err = c.FinishStage(missionId, "stage-2", false)
  if err != nil {
    t.Fatalf(`Could not finish stage`)
  }
  if !r.IsComplete {
    t.Fatalf(`Mission should be complete`)
  }
}

func TestAPI_SavePlan(t *testing.T) {
  c := client.New("test", "")
  err := c.SavePlan("tests/test_plan.json")
  if err != nil {
    t.Fatalf(`Plan should be saved without error`)
  }

  err = c.SavePlan("tests/test_plan.json") // try to save the save plan again
  if err != nil {
    t.Fatalf(`Couldn't update saved plan`)
  }

  res, err := c.CreateMission("test-plan", "")
  if err != nil {
    t.Fatalf(`Couldn't create mission with newly saved plan`)
  }

  m, err := c.GetMission(res.Id)
  if err != nil {
    t.Fatalf(`Couldn't get mission`)
  }

  stageCount := 0
  for _, stage := range m.Stages {
    if stage.Name == "stage-1" || stage.Name == "stage-2" {
      stageCount += 1
    }
  }
  if stageCount != 2 {
    t.Fatalf(`Number of stages in the mission did not match the saved plan`)
  }

  _, err = c.StartStage(res.Id, "stage-1", false)
  if err != nil {
    t.Fatalf(`Couldn't start stage of mission from saved plan`)
  }

  plans, err := c.ListPlans()
  planExists := false
  for i := range plans {
    if plans[i] == "test-plan" {
      planExists = true
    }
  }
  if !planExists {
    t.Fatalf("Plan not found when listing plans")
  }

}

func TestAPI_DeletePlan(t *testing.T) {
  c := client.New("test", "")
  c.SavePlan("tests/test_plan_deleted.json")

  p, err := c.GetPlan("test-plan-deleted")
  if err != nil {
    t.Fatalf(`Couldn't get saved plan`)
  }
  if p.Name != "test-plan-deleted" && len(p.Stages) != 2 {
    t.Fatalf(`Client.GetPlan didn't return the right plan`)
  }

  err = c.DeletePlan("test-plan-deleted")
  if err != nil {
    t.Fatalf(`Couldn't delete saved plan`)
  }

  _, err = c.CreateMission("test-plan-deleted", "")
  if err == nil {
    t.Fatalf(`Created mission with deleted plan`)
  }

  plans, err := c.ListPlans()
  if err != nil {
    t.Fatalf("Got an error when trying to list plans")
  }
  planExists := false
  for i := range plans {
    if plans[i] == "test-plan-deleted" {
      planExists = true
    }
  }
  if planExists {
    t.Fatalf("Deleted plan still exists when listing plans")
  }

}

func TestAPI_UsePassword(t *testing.T) {
  a := New("")
  a.config.Port = "8001"

  // attempting to use a short password should result in an error
  err := a.SetPassword("foobar")
  if err == nil {
    t.Fatalf("Did not get an error when using a short password")
  }
  err = a.SetPassword("foobar1 234")
  if err == nil {
    t.Fatalf("Did not get an error when using a password with invalid characters")
  }

  err = a.SetPassword("foobar1234")
  if err != nil {
    t.Fatalf("Got an error setting a valid password.")
  }

  go a.Run()

  // use password to create a key
  c := client.New("", "http://localhost:8001/api/v1")
  _, err = c.CreateKey("", "TestAPI_UsePassword", "foobar1234")
  if err != nil {
    t.Fatalf("Could not create key in password protected API")
  }

  _, err = c.CreateKey("", "TestAPI_UsePassword", "wrongpassword")
  if err == nil {
    t.Fatalf("Using the wrong password should give an error")
  }

  _, err = c.CreateKey("", "TestAPI_UsePassword", "")
  if err == nil {
    t.Fatalf("Using no password should give an error")
  }
}

// createAConflict attempts to update the mission and returns any errors. There should be 429 errors if this function
// is run multiple times at the same time, but the client should retry for them until the stages complete successfully.
func createAConflict(client *client.Client, missionId string, stage rune, errorChannel chan error) {
  _, err := client.StartStage(missionId, "s1"+string(stage), false)
  errorChannel <- err
  _, err = client.FinishStage(missionId, "s1"+string(stage), false)
  errorChannel <- err
  _, err = client.StartStage(missionId, "s2"+string(stage), false)
  errorChannel <- err
  _, err = client.FinishStage(missionId, "s2"+string(stage), false)
  errorChannel <- err
  fmt.Println("finished all")
}

// note: this tests takes longer because the client has to retry!
func TestAPI_ConcurrentMissionUpdates(t *testing.T) {

  c := client.New("test", "")
  res, err := c.CreateMission("tests/test_plan_big.json", "ConcurrentMissionUpdates")
  if err != nil {
    panic(err)
  }
  errorChannel := make(chan error)
  c.StartStage(res.Id, "s0", false)
  c.FinishStage(res.Id, "s0", false)
  for _, letter := range "abcdefghij" {
    go createAConflict(&c, res.Id, letter, errorChannel)
  }

  gotConflict := false
  counter := 0
  for err := range errorChannel {
    counter += 1

    if err != nil {
      gotConflict = true
      fmt.Println(err.Error())
    }
    if counter >= 10*4 {
      close(errorChannel)
    }
  }

  if gotConflict {
    t.Fatalf("A stage failed!")
  }

  _, err = c.StartStage(res.Id, "s3", false)
  if err != nil {
    t.Fatalf("Couldn't start final stage (this means concurrent stages failed)")
  }
  _, err = c.FinishStage(res.Id, "s3", false)
  if err != nil {
    t.Fatalf("Couldn't start final stage (this means concurrent stages failed)")
  }
}

// test completed mission + active missions + delete mission
func TestAPI_CompletedMissions(t *testing.T) {

  api := New("")
  api.config.Port = "8002"
  go api.Run()

  key, err := api.CreateKey("", "test-delete") // generate random key

  c := client.New(key, "http://localhost:8002/api/v1")
  completedMissions := api.CompletedMissions(key)
  if len(completedMissions) != 0 {
    t.Fatalf("New key has completed missions.")
  }

  c.SavePlan("tests/test_plan.json")
  active := api.ActiveMissions(key, "test-plan")
  if len(active) != 0 {
    t.Fatalf("New plan has active missions.")
  }

  res, err := c.CreateMission("test-plan", "CompletedMissions")
  if err != nil {
    panic(err)
  }
  active = api.ActiveMissions(key, "test-plan")
  if len(active) != 1 || active[0] != res.Id {
    t.Fatalf("Active mission is not listed as an active mission.")
  }

  // complete the mission
  c.StartStage(res.Id, "stage-1", false)
  c.FinishStage(res.Id, "stage-1", false)
  c.SkipStage(res.Id, "stage-2")

  completedMissions = api.CompletedMissions(key)
  if len(completedMissions) != 1 || completedMissions[0] != res.Id {
    t.Fatalf("Completed mission is not listed as a completed mission.")
  }

  deleteRes, err := c.DeleteMission(res.Id)
  if err != nil {
    t.Fatalf("Got an error deleting mission.")
  }
  if deleteRes.Id != res.Id {
    t.Fatalf("Deleted mission ID doesn't match.")
  }

  active = api.ActiveMissions(key, "test-plan")
  if len(active) != 0 {
    t.Fatalf("Deleted mission is still listed as an active mission.")
  }
  completedMissions = api.CompletedMissions(key)
  if len(completedMissions) != 0 {
    t.Fatalf("Deleted mission is still listed as a completed mission.")
  }

  api.DeleteKey(key)
}
