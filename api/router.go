package api

import (
	"encoding/json"
	"github.com/datasparq-ai/houston/model"
	"github.com/gorilla/mux"
	"net/http"
)

// GetStatus godoc
// @Summary Get API status.
// @Description Check that the API is available and healthy.
// @ID status
// @Success 200 {object} model.Success
// @Failure 500 {object} model.Error
// @Router /api/v1 [get]
func (a *API) GetStatus(w http.ResponseWriter, r *http.Request) {
	payload, _ := json.Marshal(model.Success{Message: "all systems green"})

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}

// @title Houston API
// @version 1.0
// @description Workflow Orchestration API. You can visit the GitHub repository at https://github.com/datasparq-ai/houston

// @contact.name Matt Simmons
// @contact.email info@datasparq.ai

// @license.name GNU General Public License version 3
// @license.url  https://github.com/datasparq-ai/houston/blob/main/LICENSE

// @host localhost:8000
// @BasePath /api/v1
func (a *API) initRouter() {

	router := mux.NewRouter().StrictSlash(true)
	router.Use(rateLimit)
	router.Use(logRequest)
	router.Use(recovery)
	go limiter.CleanUpIPs()

	router.HandleFunc("/api/v1", a.GetStatus).Methods("GET")

	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	//apiRouter.Use(rateLimit)
	//apiRouter.Use(loggingMiddleware)
	apiRouter.Use(a.checkKey)
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
	apiRouter.HandleFunc("/missions/{id}", a.deleteMission).Methods("DELETE")
	apiRouter.HandleFunc("/completed", a.GetCompletedMissions).Methods("GET")
	apiRouter.HandleFunc("/logs", a.GetLogs).Methods("GET")

	// note: a user can get the name of a key without the admin password, provided they have the key
	apiRouter.HandleFunc("/key", a.GetKey).Methods("GET")
	apiRouter.HandleFunc("/key/{key}", a.GetKeyWebhook).Methods("GET").
		MatcherFunc(func(r *http.Request, match *mux.RouteMatch) bool {
			// prevent this handler from running in place of the ListKeys handler
			return r.URL.Path != "/api/v1/key/all"
		})

	apiKeyRouter := router.PathPrefix("/api/v1/key").Subrouter()
	apiKeyRouter.Use(a.checkAdminPassword)
	apiKeyRouter.HandleFunc("/all", a.ListKeys).Methods("GET")
	apiKeyRouter.HandleFunc("", a.deleteKey).Methods("DELETE")
	apiKeyRouter.HandleFunc("", a.PostKey).Methods("POST")
	//apiKeyRouter.HandleFunc("/password", a.PostPassword).Methods("GET")  // TODO: route to change password

	a.router = router

	log.Debug("Router initialised")

}
