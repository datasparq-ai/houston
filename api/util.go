package api

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"github.com/datasparq-ai/houston/mission"
	"github.com/datasparq-ai/houston/model"
)

// reservedKeys can't be used as mission names or keys
var reservedKeys = []string{"u", "n", "a", "c", "m"}

// letters contains all characters that can be used in generated API keys and the randomly generated salt
var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// these characters are not allowed in keys, plan names or mission IDs
var disallowedCharacters = "| ,\n\r\t%&<>{}[]\\?;\"'`"

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
	keyLog.Error(err)

	res := model.Error{
		Message: err.Error(),
		Type:    strings.Replace(fmt.Sprintf("%T", err), "*", "", 1),
		Code:    model.ErrorCode(err),
	}

	payload, _ := json.Marshal(res)

	w.WriteHeader(res.Code)
	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}
