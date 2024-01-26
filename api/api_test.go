/**
Tests for the API. Each test creates a new API instance, key, and mission with the ID set to the name of the test.
There are many more tests for API functionality in main_test.go, which tests both the API and the client in conjunction.
*/

package api

import (
	"encoding/json"
	"github.com/datasparq-ai/houston/model"
	"os"
	"testing"
)

func TestAPI_CreateKey(t *testing.T) {

	a := New("")
	key, err := a.CreateKey("", "") // generate random key
	if err != nil {
		t.Fatalf(`Could not create key`)
	}
	if key == "" {
		t.Fatalf(`Random key not created; key is empty string`)
	}

	key2, _ := a.CreateKey("", "")
	if key == key2 {
		t.Fatalf(`Key is not random`)
	}

	err = a.DeleteKey(key) // clean up
	if err != nil {
		t.Fatalf(`Could not delete key`)
	}
}

// test completed mission + active missions + delete mission
func TestAPI_CompletedMissions(t *testing.T) {

	api := New("")

	key, err := api.CreateKey("", "test-delete") // generate random key

	completedMissions := api.CompletedMissions(key)
	if len(completedMissions) != 0 {
		t.Fatalf("New key has completed missions.")
	}

	planBytes, _ := os.ReadFile("../tests/test_plan.json")
	var plan model.Plan
	json.Unmarshal(planBytes, &plan)

	if plan.Name != "test-plan" {
		t.Fatalf("Failed to load plan")
	}

	err = api.SavePlan(key, plan)
	if err != nil {
		t.Fatalf("Failed to save plan")
	}

	active := api.ActiveMissions(key, "test-plan")
	if len(active) != 0 {
		t.Fatalf("New plan has active missions.")
	}

	missionId, err := api.CreateMissionFromPlan(key, "test-plan", "", map[string]interface{}{"foo": "bar", "d": map[string]interface{}{"fp": 3}})
	if err != nil {
		t.Fatalf("Failed to start a mission")
	}

	active = api.ActiveMissions(key, "test-plan")
	if len(active) != 1 || active[0] != missionId {
		t.Fatalf("Active mission is not listed as an active mission.")
	}

	api.UpdateStageState(key, missionId, "stage-1", "started", false)
	api.UpdateStageState(key, missionId, "stage-1", "finished", false)
	api.UpdateStageState(key, missionId, "stage-2", "skipped", false)

	completedMissions = api.CompletedMissions(key)
	if len(completedMissions) != 1 || completedMissions[0] != missionId {
		t.Fatalf("Completed mission is not listed as a completed mission.")
	}

	api.DeleteMission(key, missionId)

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
