/*
API routes for plans

*/

package main

import (
	"encoding/json"
	"fmt"
	"github.com/datasparq-ai/houston/model"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"strings"
)

// GetPlan godoc
// @Summary Gets plan given its name.
// @Description Returns the plan definition as JSON. If the plan was never explicitly saved then it will return 404.
// @ID get-plan
// @Tags Plan
// @Param Header header string true "Houston Key"
// @Param name path string true "The name of the plan"
// @Success 200 {object} model.Success
// @Failure 404,500 {object} model.Error
// @Router /v1/plans/{name} [get]
func (a *API) GetPlan(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	planName := vars["name"]
	key := r.Header.Get("x-access-key") // key has been checked by checkKey middleware

	plan, ok := a.db.Get(key, "p|"+planName)

	if !ok {
		handleError(&model.PlanNotFoundError{PlanName: planName}, w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(plan))
}

// GetPlanAsMission godoc
// @Summary Gets plan in format of a mission.
// @Description This is identical to GetPlan but returns the plan in the same format as a mission
// @ID get-plan-as-mission
// @Tags Plan
// @Param Header header string true "Houston Key"
// @Param name path string true "The name of the plan"
// @Success 200 {object} model.Success
// @Failure 404,500 {object} model.Error
// @Router /v1/plans/{name}/m [get]
func (a *API) GetPlanAsMission(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	planName := vars["name"]
	key := r.Header.Get("x-access-key") // key has been checked by checkKey middleware

	plan, ok := a.db.Get(key, "p|"+planName)

	if !ok {
		handleError(&model.PlanNotFoundError{PlanName: planName}, w)
		return
	}

	var p model.Plan
	err := json.Unmarshal([]byte(plan), &p)
	if err != nil {
		// plan was improperly formatted (should be impossible as plans are checked before they're saved)
		handleError(fmt.Errorf("couldn't return plan as plan was invalid"), w)
	}
	planAsMission := NewMissionFromPlan(&p)
	planBytes, _ := json.Marshal(planAsMission)

	w.Header().Set("Content-Type", "application/json")
	w.Write(planBytes)
}

// PostPlan godoc
// @Summary Updates the state of a stage in an in-progress mission.
// @Description This route is transactional, meaning it will fail and result in 429 response if the same mission is currently being modified.
// @ID post-plan
// @Tags Plan
// @Param Header header string true "Houston Key"
// @Param Body body model.Plan true "The id, services, stages and parameters of a plan."
// @Success 200 {object} model.Success
// @Failure 404,500 {object} model.Error
// @Router /v1/plans [post]
func (a *API) PostPlan(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := io.ReadAll(r.Body)
	var plan model.Plan
	err := json.Unmarshal(reqBody, &plan)
	if err != nil {
		// TODO: helpful error message for json validation errors, e.g. 'state is missing'
		handleError(err, w)
		return
	}
	key := r.Header.Get("x-access-key") // key has been checked by checkKey middleware
	err = a.SavePlan(key, plan)
	if err != nil {
		handleError(err, w)
		return
	}
}

// DeletePlan godoc
// @Summary Deletes a plan and all its missions from the database.
// @Description Deletes a plan and associated missions given its name. Any missions in progress will be deleted.
// @ID delete-plan
// @Tags Plan
// @Param Header header string true "Houston Key"
// @Param Body body model.Plan true "The id, services, stages and parameters of a plan."
// @Param name path string true "The name of the plan"
// @Success 200 {object} model.Success
// @Failure 404,500 {object} model.Error
// @Router /v1/plans/{name} [delete]
func (a *API) DeletePlan(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	planName := vars["name"]
	key := r.Header.Get("x-access-key") // key has been checked by checkKey middleware

	wasDeleted := a.db.Delete(key, "p|"+planName)

	activeMissions, ok := a.db.Get(key, "a|"+planName)
	if ok {
		// delete missions
		for _, missionId := range strings.Split(activeMissions, ",") {
			a.db.Delete(key, missionId)
		}
		// delete missions from the completed list
		completedList := a.CompletedMissions(key)
		var newCompletedList []string
	Loop:
		for _, completedMissionId := range completedList {
			for _, missionId := range strings.Split(activeMissions, ",") {
				if missionId == completedMissionId {
					continue Loop
				}
			}
			newCompletedList = append(newCompletedList, completedMissionId)
		}

		completedListBytes := strings.Join(newCompletedList, ",")
		a.db.Set(key, "c", completedListBytes)
	}

	wasDeleted = wasDeleted && a.db.Delete(key, "a|"+planName)

	if !wasDeleted {
		err := fmt.Errorf("could not delete plan '%v'", planName)
		handleError(err, w)
		return
	}

	a.ws <- message{key: key, Event: "planDeleted", Content: []byte(planName)}
	payload, _ := json.Marshal(model.Success{Message: "Deleted " + planName})

	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}

// GetPlans godoc
// @Summary Gets all plans
// @Description Returns a list of all existing plans given a Houston Key.
// @ID get-plans
// @Tags Plan
// @Param Header header string true "Houston Key"
// @Success 200 {object} model.Success
// @Failure 404,500 {object} model.Error
// @Router /v1/plans/ [get]
func (a *API) GetPlans(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("x-access-key") // key has been checked by checkKey middleware
	plans, err := a.ListPlans(key)
	if err != nil {
		handleError(err, w)
		return
	}
	payload, _ := json.Marshal(plans)
	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}


// GetPlanMissions godoc
// @Summary Gets a plan's missions
// @Description Returns a list of the IDs of all active (non archived) missions for the plan.
// @ID get-plan-missions
// @Tags Plan
// @Param Header header string true "Houston Key"
// @Param name path string true "The name of the plan"
// @Success 200 {object} model.Success
// @Failure 404,500 {object} model.Error
// @Router /v1/plans/{name}/missions [get]
func (a *API) GetPlanMissions(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("x-access-key") // key has been checked by checkKey middleware
	vars := mux.Vars(r)
	plan := vars["name"]
	missions := a.ActiveMissions(key, plan)

	payload, _ := json.Marshal(missions)
	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}
