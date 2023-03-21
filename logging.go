package main

import (
    "encoding/json"
	"github.com/sirupsen/logrus"
	"github.com/gorilla/mux"
	"os"
	"net/http"
	"time"
)

var log *logrus.Logger

func initLog() {
	log = logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.DebugLevel)

	// todo: get logs directory from config
	err := os.MkdirAll("logs", 0700)
	if err == nil {
		SetLoggingFile("")
	} else {
		log.SetOutput(os.Stderr)
		log.Info("Failed to log to file, using default stderr")
	}

	log.Debug("Useful debugging information.")
	log.Info("Something noteworthy happened!")
	log.Warn("You should probably take a look at this.")
	log.Error("Something failed but I'm not quitting.")
}

// SetLoggingFile switches the logging output file to a file specific to the key and the current day. If no key is
// provided then logs go to the main logging file, which is only accessible by the admin.
func SetLoggingFile(key string) {

	dt := time.Now()
	day := dt.Format("01022006")

	var logFileName string
	if key == "" {
		logFileName = "logs/api_" + day + "_log.txt"
	} else {
		logFileName = "logs/key_" + key + "_" + day + "_log.txt"
	}

	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

	if err == nil {
		log.SetOutput(file)
	} else {
		log.SetOutput(os.Stderr)
		log.Info("Failed to log to file, using default stderr")
	}
}

// GetLogs godoc
// @Summary Returns logs for the key provided.
// @Description Returns logs for the key provided. Defaults to today.
// @ID get-logs
// @Tags Logs
// @Param x-access-key header string true "Houston Key"
// @Param logDate path string true "Date of logs required in format MMDDYYYY"
// @Success 200 {object} ???
// @Failure 404,500 {object} model.Error
// @Router /api/v1/logs [get]
func (a *API) GetLogs(w http.ResponseWriter, r *http.Request) {
    key := r.Header.Get("x-access-key") // key has been checked by checkKey middleware
    vars := mux.Vars(r)
    logDate := vars["logDate"]
    var requiredLogDate string

    if logDate == "" {
        dtToday := time.Now()
        today := dtToday.Format("01022006")
        requiredLogDate = today
    } else {
        requiredLogDate = logDate
    }

    var logFileName string
    logFileName = "logs/key_" + key + "_" + requiredLogDate + "_log.txt"
    contents, err := os.ReadFile(logFileName)
    if err != nil {
		handleError(err, w)
	}
	var logs string
	logs = string(contents)


    payload, _ := json.Marshal(logs)
	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}
