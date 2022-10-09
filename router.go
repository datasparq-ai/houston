package main

import (
  "encoding/json"
  "fmt"
  "github.com/datasparq-ai/houston/mission"
  "github.com/datasparq-ai/houston/model"
  "github.com/gorilla/mux"
  "io/ioutil"
  "net/http"
  "strings"
)

func handleError(err error, w http.ResponseWriter) {
  res := model.Error{Message: err.Error()}
  fmt.Println("ERROR:", res.Message)
  payload, _ := json.Marshal(res)
  switch err.(type) {
  case *model.TransactionFailedError:
    w.WriteHeader(http.StatusTooManyRequests)
  case *model.KeyNotFoundError, *model.PlanNotFoundError:
    w.WriteHeader(http.StatusNotFound)
  default:
    w.WriteHeader(http.StatusBadRequest)
  }
  w.Header().Set("Content-Type", "application/json")
  w.Write(payload)
}

// GetStatus can be used to check that the API is available
func (a *API) GetStatus(w http.ResponseWriter, r *http.Request) {
  payload, _ := json.Marshal(map[string]string{"message": "all systems green"})

  w.WriteHeader(http.StatusOK)
  w.Header().Set("Content-Type", "application/json")
  w.Write(payload)
}

func (a *API) GetMission(w http.ResponseWriter, r *http.Request) {
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

// GetPlanMissions returns a list of the IDs of all active (non archived) missions for the plan
func (a *API) GetPlanMissions(w http.ResponseWriter, r *http.Request) {
  key := r.Header.Get("x-access-key") // key has been checked by checkKey middleware
  vars := mux.Vars(r)
  plan := vars["name"]
  missions := a.ActiveMissions(key, plan)

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
  reqBody, _ := ioutil.ReadAll(r.Body)
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

  w.Header().Set("Content-Type", "application/json")
  w.Write(payload)
}

// PostMissionStage updates the state of a stage in an in-progress mission.
// This route is transactional, meaning it will fail and result in 429 response if the same mission is currently being
// modified.
func (a *API) PostMissionStage(w http.ResponseWriter, r *http.Request) {

  reqBody, _ := ioutil.ReadAll(r.Body)
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
  payload, _ := json.Marshal(res)
  w.Header().Set("Content-Type", "application/json")
  w.Write(payload)

}

// GetCompletedMissions returns a list of the IDs of all completed (non archived) missions.
// These missions will also be in the list returned by GetMissions.
// This list is stored in a separate redis key for performance reasons.
// Completed missions should be deleted after being archived by the user to minimise the amount of storage required by
// the database.
func (a *API) GetCompletedMissions(w http.ResponseWriter, r *http.Request) {
  key := r.Header.Get("x-access-key") // key has been checked by checkKey middleware
  missions := a.CompletedMissions(key)
  payload, _ := json.Marshal(missions)
  w.Header().Set("Content-Type", "application/json")
  w.Write(payload)
}

func (a *API) PostPlan(w http.ResponseWriter, r *http.Request) {
  reqBody, _ := ioutil.ReadAll(r.Body)
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

// DeletePlan deletes a plan and all its missions from the database
func (a *API) DeletePlan(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  planName := vars["name"]
  key := r.Header.Get("x-access-key") // key has been checked by checkKey middleware

  wasDeleted := a.db.Delete(key, "p|"+planName)

  // TODO: delete active missions, and delete active key

  if !wasDeleted {
    // TODO: what possible reasons are there for this error? Will it ever happen?
    err := fmt.Errorf("could not delete plan '%v'", planName)
    handleError(err, w)
    return
  }
}

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

func (a *API) GetPlans(w http.ResponseWriter, r *http.Request) {
  key := r.Header.Get("x-access-key") // key has been checked by checkKey middleware
  plans, err := a.ListPlans(key)
  if err != nil {
    handleError(err, w)
    return
  }
  payload, _ := json.Marshal(plans)
  w.Header().Set("Content-Type", "application/json")
  w.Write(payload) // TODO: this is a different format to existing API, but will only affect dashboard?
}

// checkKey runs before requests that require a key to check that the key exists
func (a *API) checkKey(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    key := r.Header.Get("x-access-key")
    if key == "" {
      err := fmt.Errorf("key not provided") // TODO: make error type
      handleError(err, w)
      return
    }
    // check that key exists
    _, ok := a.db.Get(key, "u")
    if !ok {
      err := &model.KeyNotFoundError{}
      handleError(err, w)
      return
    }

    next.ServeHTTP(w, r)
  })
}

// checkAdminPassword runs before all admin routes
func (a *API) checkAdminPassword(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if a.config.Password == "" {
      next.ServeHTTP(w, r) // if API is not password protected - do nothing
      return
    }

    username, password, ok := r.BasicAuth()
    if !ok {
      err := fmt.Errorf("admin password not provided") // TODO: make error type
      handleError(err, w)
      return
    }
    if username != "admin" {
      err := fmt.Errorf("username provided was not found") // TODO: make error type
      handleError(err, w)
      return
    }
    // check that password matches hash of password stored in config
    if a.config.Password == hashPassword(password, a.config.Salt) {
      next.ServeHTTP(w, r)
      return
    } else {
      err := fmt.Errorf("incorrect password") // TODO: make error type
      handleError(err, w)
      return
    }
  })
}

// PostKey is used to create a new key. The request does not need to contain a body if a random key should be generated.
// the newly created key is returned as bytes.
func (a *API) PostKey(w http.ResponseWriter, r *http.Request) {
  var key model.Key

  reqBody, _ := ioutil.ReadAll(r.Body)

  if len(reqBody) == 0 {
    key = model.Key{}
  } else {
    err := json.Unmarshal(reqBody, &key)
    if err != nil {
      // TODO: create error type for invalid request with helpful message
      handleError(err, w)
      return
    }
  }

  keyId, err := a.CreateKey(key.Id, key.Name)
  if err != nil {
    handleError(err, w)
    return
  }
  w.Write([]byte(keyId))
}

func (a *API) initRouter() {

  router := mux.NewRouter().StrictSlash(true)
  router.HandleFunc("/api/v1", a.GetStatus).Methods("GET")

  apiRouter := router.PathPrefix("/api/v1").Subrouter()
  apiRouter.Use(a.checkKey)
  //apiRouter.HandleFunc("/plans/{name}", a.PutPlan).Methods("PUT")
  apiRouter.HandleFunc("/plans/", a.GetPlans).Methods("GET")
  apiRouter.HandleFunc("/plans", a.PostPlan).Methods("POST")
  apiRouter.HandleFunc("/plans/{name}/missions", a.GetPlanMissions).Methods("GET")
  apiRouter.HandleFunc("/plans/{name}", a.GetPlan).Methods("GET")
  apiRouter.HandleFunc("/plans/{name}", a.DeletePlan).Methods("DELETE")
  apiRouter.HandleFunc("/missions/", a.GetMissions).Methods("GET")
  apiRouter.HandleFunc("/missions", a.PostMission).Methods("POST")
  apiRouter.HandleFunc("/missions/{id}/stages/{name}", a.PostMissionStage).Methods("POST")
  apiRouter.HandleFunc("/missions/{id}", a.GetMission).Methods("GET")
  apiRouter.HandleFunc("/missions/{id}/report", a.GetMissionReport).Methods("GET")
  apiRouter.HandleFunc("/missions/{id}", a.DeleteMission).Methods("DELETE")
  apiRouter.HandleFunc("/completed", a.GetCompletedMissions).Methods("GET")

  //apiKeyRouter.HandleFunc("/password", a.PostPassword).Methods("GET")  // TODO: route to change password

  apiKeyRouter := router.PathPrefix("/api/v1/key").Subrouter()
  apiKeyRouter.Use(a.checkAdminPassword)
  //apiKeyRouter.HandleFunc("", a.ListKeys).Methods("GET")  // TODO: check password
  //apiKeyRouter.HandleFunc("/{id}", a.GetKey).Methods("GET")  // TODO: check key (get description)
  //apiKeyRouter.HandleFunc("/{id}", a.DeleteKey).Methods("DELETE")  // TODO
  apiKeyRouter.HandleFunc("", a.PostKey).Methods("POST")

  a.router = router

}
