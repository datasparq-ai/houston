package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/term"
)

var dateLayout = "20060102"

// log is used for logs that only the API server admin should be able to see
var log *logrus.Logger

// keyLog is used for logs relating to a specific API key. These are viewable by anyone with that API key via the logs viewer UI
var keyLog *logrus.Logger
var isTerminal bool

func initLog() {
	isTerminal = term.IsTerminal(int(os.Stdout.Fd()))

	log = logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.DebugLevel)

	keyLog = logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.DebugLevel)

	// todo: get logs directory from config
	err := os.MkdirAll("logs", 0700)
	if err == nil {
		SetLoggingFile(log, "")
		SetLoggingFile(keyLog, "")
		log.Info("Logging started successfully")
	} else {
		log.SetOutput(os.Stderr)
		keyLog.SetOutput(os.Stderr)
		log.Info("Failed to log to file, using default stderr")
	}

	// log.Debug("Useful debugging information.")
	// log.Info("Something noteworthy happened!")
	// log.Warn("You should probably take a look at this.")
	// log.Error("Something failed but I'm not quitting.")
}

// SetLoggingFile switches the logging output file to a file specific to the key and the current day. If no key is
// provided then logs go to the main logging file, which is only accessible by the admin.
func SetLoggingFile(logger *logrus.Logger, key string) {

	day := time.Now().Format(dateLayout)

	var logFileName string
	if key == "" {
		logFileName = "logs/api_" + day + "_log.txt"
	} else {
		logFileName = "logs/key_" + key + "_" + day + "_log.txt"
	}

	file, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

	if err == nil {
		if key == "" {
			// If in interactive terminal, stdout should only be used for print statements and logs only written to file
			if isTerminal {
				logger.SetOutput(file)
			} else {
				mw := io.MultiWriter(os.Stdout, file)
				logger.SetOutput(mw)
			}
		} else {
			logger.SetOutput(file)
		}
	} else {
		logger.SetOutput(os.Stderr)
		logger.Info("Failed to log to file, using default stderr")
	}
}

// GetLogs godoc
// @Summary Returns logs for the key provided.
// @Description Returns logs for the key provided. Defaults to today.
// @ID get-logs
// @Tags Logs
// @Param x-access-key header string true "Houston Key"
// @Param date query string false "Date of logs required in format YYYYMMDD"
// @Success 200 {string} string
// @Failure 404,500 {object} model.Error
// @Router /api/v1/logs [get]
func (a *API) GetLogs(w http.ResponseWriter, r *http.Request) {

	key := r.Header.Get("x-access-key") // key has been checked by checkKey middleware
	logDate := r.URL.Query().Get("date")

	if logDate == "" {
		today := time.Now().Format(dateLayout)
		logDate = today
	}
	// check that date is valid: look for invalid characters and check length
	if strings.ContainsAny(logDate, disallowedCharacters+"/~.-_:") || len(logDate) != 8 {
		err := fmt.Errorf("invalid log query: '%v' is not a valid date; dates must use YYYYMMDD format", logDate)
		handleError(err, w)
		return
	}

	logFileName := "logs/key_" + key + "_" + logDate + "_log.txt"
	contents, err := os.ReadFile(logFileName)

	if err != nil {
		switch err.(type) {
		case *fs.PathError:
			// if so file exists for the key and date then return code 404 and no data
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte{})
		default:
			handleError(err, w)
		}
		return
	}
	var logs string
	logs = string(contents)

	payload, _ := json.Marshal(logs)
	w.Header().Set("Content-Type", "text/plain")
	w.Write(payload)
}
