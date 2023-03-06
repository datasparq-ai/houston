package main

import (
	"encoding/json"
	"fmt"
	"github.com/datasparq-ai/houston/mission"
	"github.com/datasparq-ai/houston/model"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func (a *API) GetMission(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	missionId := vars["id"]
	key := r.Header.Get("x-access-key") // key has been checked by checkKey middleware
	missionString, ok := a.db.Get(key, missionId)

	if !ok {
		err := fmt.Errorf("mission with id '%v' not found", missionId)
		handleError(err, w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(missionString))
}

func (a *API) GetMissionReport(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	missionId := vars["id"]
	key := r.Header.Get("x-access-key") // key has been checked by checkKey middleware
	missionString, ok := a.db.Get(key, missionId)
	if !ok {
		// TODO: mission not found error
		err := fmt.Errorf("mission with id '%v' not found", missionId)
		handleError(err, w)
		return
	}
	m, err := mission.NewFromJSON([]byte(missionString))
	if err != nil {
		handleError(err, w)
	}

	report := m.Report()

	payload, _ := json.Marshal(model.Success{Message: report})
	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}

// GetMissions returns a list of the IDs of all active (non archived) missions
func (a *API) GetMissions(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("x-access-key") // key has been checked by checkKey middleware
	missions, err := a.AllActiveMissions(key)
	if err != nil {
		handleError(err, w)
		return
	}
	payload, _ := json.Marshal(missions)
	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}

// PostMission godoc
// @Summary Creates a new mission and returns the ID.
// @Description Creates a new mission using the ID provided or with an automatically generated ID if none is provided.
// @ID create-mission
// @Tags Mission
// @Param Header x-access-key string true "Houston key"
// @Param Body body model.MissionCreateRequest true "The plan, ID, and parameters to give to the new mission."
// @Success 200 {object} model.MissionCreatedResponse
// @Failure 404,500 {object} model.Error
// @Router /v1/missions [post]
func (a *API) PostMission(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := io.ReadAll(r.Body)
	var mission model.MissionCreateRequest
	err := json.Unmarshal(reqBody, &mission)
	if err != nil {
		handleError(err, w)
		return
	}

	key := r.Header.Get("x-access-key") // key has been checked by checkKey middleware

	newMissionId, err := a.CreateMissionFromPlan(key, mission.Plan, mission.Id)
	if err != nil {
		handleError(err, w)
		return
	}
	res := model.MissionCreatedResponse{Id: newMissionId}

	payload, _ := json.Marshal(res)
	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)

}

func (a *API) DeleteMission(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	missionId := vars["id"]
	key := r.Header.Get("x-access-key") // key has been checked by checkKey middleware

	missionString, ok := a.db.Get(key, missionId)
	if !ok {
		return
	}
	var m model.Mission
	// there is unlikely to be an error here, but if there is just skip removing mission from active list
	err := json.Unmarshal([]byte(missionString), &m)
	if err == nil {
		// remove from active missions
		activeStr, _ := a.db.Get(key, "a|"+m.Name)
		activeStr = strings.Replace(","+activeStr+",", ","+missionId+",", "", 1)
		activeStr = strings.Trim(activeStr, ",")
		a.db.Set(key, "a|"+m.Name, activeStr)
	}

	// remove from completed missions
	completeString, ok := a.db.Get(key, "c")
	completeString = strings.Replace(","+completeString+",", ","+missionId+",", "", 1)
	completeString = strings.Trim(completeString, ",")
	a.db.Set(key, "c", completeString)

	// delete mission
	a.db.Delete(key, missionId)

	payload, _ := json.Marshal(model.Success{Message: "Deleted " + missionId})

	a.ws <- message{key: key, Event: "missionDeleted", Content: []byte(missionId)}

	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}

// PostMissionStage updates the state of a stage in an in-progress mission.
// This route is transactional, meaning it will fail and result in 429 response if the same mission is currently being
// modified.
func (a *API) PostMissionStage(w http.ResponseWriter, r *http.Request) {

	reqBody, _ := io.ReadAll(r.Body)
	var stage model.MissionStageStateUpdate
	err := json.Unmarshal(reqBody, &stage)
	if err != nil {
		// TODO: helpful error message for json validation errors, e.g. 'state is missing'
		handleError(err, w)
		return
	}

	vars := mux.Vars(r)
	missionId := vars["id"]
	stageName := vars["name"]
	key := r.Header.Get("x-access-key") // key has been checked by checkKey middleware

	res, err := a.UpdateStageState(key, missionId, stageName, stage.State, stage.IgnoreDependencies)
	if err != nil {
		handleError(err, w)
		return
	}

	// increment usage - ignore all errors
	usage, _ := a.db.Get(key, "u")
	intUsage, _ := strconv.Atoi(usage)
	a.db.Get(key, fmt.Sprintf("%v", intUsage+1))

	payload, _ := json.Marshal(res)
	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)

}
