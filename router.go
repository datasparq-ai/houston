package main

import (
  "encoding/json"
  "github.com/gorilla/mux"
  "net/http"
)

// GetStatus can be used to check that the API is available
func (a *API) GetStatus(w http.ResponseWriter, r *http.Request) {
  payload, _ := json.Marshal(map[string]string{"message": "all systems green"})

  w.WriteHeader(http.StatusOK)
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

// @title Houston API
// @version 1.0
// @description Houston API documentation. You can visit the GitHub repository at https://github.com/datasparq-ai/houston

// @contact.name Matt Simmons
// @contact.email info@datasparq.ai

// @license.name Houston: Workflow Orchestration API
// @license.url  https://github.com/datasparq-ai/houston/blob/main/LICENSE

// @host localhost:8000
// @BasePath /api/v1
func (a *API) initRouter() {

  router := mux.NewRouter().StrictSlash(true)
  router.Use(rateLimit)
  go limiter.CleanUpIPs()

  router.HandleFunc("/api/v1", a.GetStatus).Methods("GET")

  apiRouter := router.PathPrefix("/api/v1").Subrouter()
  apiRouter.Use(a.checkKey)
  //apiRouter.HandleFunc("/plans/{name}", a.PutPlan).Methods("PUT")
  apiRouter.HandleFunc("/plans/", a.GetPlans).Methods("GET")
  apiRouter.HandleFunc("/plans", a.PostPlan).Methods("POST")
  apiRouter.HandleFunc("/plans/{plan}/missions/{id}", a.GetMission).Methods("GET")
  apiRouter.HandleFunc("/plans/{name}/missions", a.GetPlanMissions).Methods("GET")
  apiRouter.HandleFunc("/plans/{name}/m", a.GetPlanAsMission).Methods("GET")
  apiRouter.HandleFunc("/plans/{name}", a.GetPlan).Methods("GET")
  apiRouter.HandleFunc("/plans/{name}", a.DeletePlan).Methods("DELETE")
  apiRouter.HandleFunc("/missions/", a.GetMissions).Methods("GET")
  apiRouter.HandleFunc("/missions", a.PostMission).Methods("POST")
  apiRouter.HandleFunc("/missions/{id}/stages/{name}", a.PostMissionStage).Methods("POST")
  apiRouter.HandleFunc("/missions/{id}", a.GetMission).Methods("GET")
  apiRouter.HandleFunc("/missions/{id}/report", a.GetMissionReport).Methods("GET")
  apiRouter.HandleFunc("/missions/{id}", a.DeleteMission).Methods("DELETE")
  apiRouter.HandleFunc("/completed", a.GetCompletedMissions).Methods("GET")

  // note: a user can get the name of a key without the admin password, provided they have the key
  apiRouter.HandleFunc("/key", a.GetKey).Methods("GET")

  apiKeyRouter := router.PathPrefix("/api/v1/key").Subrouter()
  apiKeyRouter.Use(a.checkAdminPassword)
  //apiKeyRouter.HandleFunc("", a.ListKeys).Methods("GET")
  //apiKeyRouter.HandleFunc("/{id}", a.DeleteKey).Methods("DELETE")  // TODO
  apiKeyRouter.HandleFunc("", a.PostKey).Methods("POST")
  //apiKeyRouter.HandleFunc("/password", a.PostPassword).Methods("GET")  // TODO: route to change password

  a.router = router

}
