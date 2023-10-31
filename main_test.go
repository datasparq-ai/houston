/**
Tests for the API and Client. Runs a local Houston API server in a goroutine and uses Houston's Go client to run tests.
Each test creates a new mission, with the ID set to the name of the test.
*/

package main

import (
	"encoding/json"
	"fmt"
	"github.com/datasparq-ai/houston/api"
	"github.com/datasparq-ai/houston/client"
	"github.com/datasparq-ai/houston/model"
	"os"
	"strings"
	"testing"
	"time"
)

var testKeyId = "test"

// creates a local server that is used by most of the tests in this file
func TestMain(m *testing.M) {

	a := api.New("")
	a.DeleteKey(testKeyId)
	a.CreateKey(testKeyId, "unittest-1")

	go a.Run()

	code := m.Run() // run tests

	a.DeleteKey(testKeyId) // clean up
	os.Exit(code)
}

// create a mission using the client
func Test_GetMission(t *testing.T) {

	c := client.New(testKeyId, "")

	data, _ := os.ReadFile("tests/test_plan.json")
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

func Test_PostMissionStage(t *testing.T) {

	c := client.New(testKeyId, "")

	data, _ := os.ReadFile("tests/test_plan.json")
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

	start := time.Now()
	_, err = c.FinishStage(missionId, "stage-1", false)
	if err != nil {
		t.Fatalf(`Could not finish stage`)
	}
	if td := time.Since(start); td > time.Millisecond*5 {
		t.Fatalf("Finishing a stage took too long at %v", td)
	}
}

func Test_PostMissionStage_StartStage_IgnoreDependencies(t *testing.T) {

	c := client.New(testKeyId, "")

	data, _ := os.ReadFile("tests/test_plan.json")
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

func Test_SavePlan(t *testing.T) {
	c := client.New(testKeyId, "")
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

func Test_DeletePlan(t *testing.T) {
	c := client.New(testKeyId, "")
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

// create keys in a password protected API
func Test_UsePassword(t *testing.T) {

	// this uses a different port and redis database (if redis is used)
	a := api.New("tests/test_config.yaml")

	go a.Run()

	// use password to create a key
	c := client.New("", "http://localhost:8001/api/v1")
	_, err := c.CreateKey("", "TestAPI_UsePassword", "foobar1234")
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
func Test_ConcurrentMissionUpdates(t *testing.T) {

	c := client.New(testKeyId, "")
	res, err := c.CreateMission("tests/test_plan_big.json", "ConcurrentMissionUpdates")
	if err != nil {
		t.Fatalf("Got an error when creating mission: %s", err.Error())
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

func Test_ListKeys(t *testing.T) {

	c := client.New(testKeyId, "")

	// Generate 2 more keys
	c.CreateKey("test2", "", "")
	c.CreateKey("test3", "", "")

	// Extract list of keys
	keys, err := c.ListKeys("")

	if err != nil {
		panic(err)
		t.Fatalf("Got an error when trying to list keys")
	}

	// note: this gives a different result when running with local db vs Redis db, because there is only one instance
	// of redis, whereas different tests use completely new instances of the local db

	// Should have the 3 keys in db including original test key
	//noKeys := len(keys)
	foundKeys := []int{0, 0, 0}
	for _, key := range keys {
		if key == testKeyId {
			foundKeys[0]++
		} else if key == "test2" {
			foundKeys[1]++
		} else if key == "test3" {
			foundKeys[2]++
		}
	}

	for _, keyCount := range foundKeys {
		if keyCount != 1 {
			t.Fatalf("A key was not listed exactly one time, got " + strings.Join(keys, ", "))
		}
	}

}

func Test_DeleteKey(t *testing.T) {
	c := client.New(testKeyId, "")

	c.DeleteKey("")

	// Extract list of keys
	keys, _ := c.ListKeys("")

	// Ensure test key is not in list of keys
	keyExists := false
	for i := range keys {
		if keys[i] == testKeyId {
			keyExists = true
		}
	}
	if keyExists {
		t.Fatalf("Deleted key still exists when listing keys")
	}
}
