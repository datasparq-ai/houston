package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/datasparq-ai/houston/mission"
	"github.com/datasparq-ai/houston/model"
	"math/rand"
	"net/http"
	"strings"
)

// reservedKeys can't be used as mission names
var reservedKeys = []string{"u", "n", "a", "c"}

// letters contains all characters that can be used in generated API keys and the randomly generated salt
var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// these characters are not allowed in plan names or mission IDs
var disallowedCharacters = []rune("| ,\n\r\t%&<>{}[]\\?;\"'`")

// used to create API keys and tokens
func createRandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func hashPassword(password, salt string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(password+salt)))
}

// NewMissionFromPlan converts a plan into the equivalent mission for graph validation purposes. Not all fields will
// be included because the mission type will not use them
// func NewMissionFromPlan(plan *model.Plan) {
func NewMissionFromPlan(plan *model.Plan) *mission.Mission {

	var stages []*mission.Stage

	for stageIdx := range plan.Stages {
		s := mission.Stage{
			Name:       plan.Stages[stageIdx].Name,
			Upstream:   plan.Stages[stageIdx].Upstream,
			Downstream: plan.Stages[stageIdx].Downstream,
			Params:     plan.Stages[stageIdx].Params,
		}
		stages = append(stages, &s)
	}

	m := mission.New(plan.Name, stages)

	return &m
}

// handleError writes an error http response given an error object
func handleError(err error, w http.ResponseWriter) {
	res := model.Error{Message: err.Error(), Type: strings.Replace(fmt.Sprintf("%T", err), "*", "", 1)}
	payload, _ := json.Marshal(res)
	switch err.(type) {
	case *model.TransactionFailedError, *model.TooManyRequestsError:
		w.WriteHeader(http.StatusTooManyRequests)
	case *model.KeyNotFoundError, *model.PlanNotFoundError:
		w.WriteHeader(http.StatusNotFound)
	case *model.BadCredentialsError:
		w.WriteHeader(http.StatusForbidden)
	default:
		w.WriteHeader(http.StatusBadRequest)

	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}
