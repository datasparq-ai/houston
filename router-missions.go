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

// GetMission godoc
// @Summary Gets mission from ID.
// @Description Gets a existing mission using the ID provided.
// @ID get-mission
// @Tags Mission
// @Param Header header string true "Houston Key"
// @Param id path string true "The id of the mission"
// @Success 200 {object} model.Success
// @Failure 404,500 {object} model.Error
// @Router /v1/missions/{id} [get]
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

// GetMissionReport godoc
// @Summary Gets a report of an existing mission.
// @Description Returns a report of an existing mission for a given Houston Key.
// @ID get-mission-report
// @Tags Mission
// @Param Header header string true "Houston Key"
// @Param id path string true "The id of the mission"
// @Success 200 {object} model.Success
// @Failure 404,500 {object} model.Error
// @Router /v1/missions/{id}/report [get]
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

// GetMissions godoc
// @Summary Gets all existing missions.
// @Description Returns all existing missions for a given Houston Key.
// @ID get-missions
// @Tags Mission
// @Param Header header string true "Houston Key"
// @Success 200 {object} model.Success
// @Failure 404,500 {object} model.Error
// @Router /v1/missions/ [get]
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
// @Param Header header string true "Houston Key"
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

// DeleteMission godoc
// @Summary Deletes mission given its ID.
// @Description Deletes any existing mission given a mission ID.
// @ID delete-mission
// @Tags Mission
// @Param Header header string true "Houston Key"
// @Param id path string true "The id of the mission"
// @Success 200 {object} model.Success
// @Failure 404,500 {object} model.Error
// @Router /v1/missions/{id} [delete]
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

// PostMissionStage godoc
// @Summary Updates the state of a stage in an in-progress mission.
// @Description This route is transactional, meaning it will fail and result in 429 response if the same mission is currently being modified.
// @ID post-mission-stage
// @Tags Mission
// @Param Header header string true "Houston Key"
// @Param Body body model.MissionStageStateUpdate true "The state of the stage and whether dependencies have been ignored."
// @Param id path string true "The id of the mission"
// @Param name path string true "The name of the plan"
// @Success 200 {object} model.Success
// @Failure 404,500 {object} model.Error
// @Router /v1/missions/{id}/stages/{name} [post]
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
