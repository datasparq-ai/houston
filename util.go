package main

import (
  "crypto/sha256"
  "fmt"
  "github.com/datasparq-ai/houston/mission"
  "github.com/datasparq-ai/houston/model"
  "math/rand"
)

var letters = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

//// nameToKey maps the full names plan attributes to the equivalent short name used in a mission.
//// Only attributes seen both in plans and missions are included.
//var nameToKey = map[string]string{
//  "name":     "n",
//  "stages":   "s",
//  "services": "a",
//}
//
//var stagesNameToKey = map[string]string{
//  "upstream":   "u",
//  "downstream": "d",
//}

// used to create API keys
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
//func NewMissionFromPlan(plan *model.Plan) {
func NewMissionFromPlan(plan *model.Plan) *mission.Mission {

  var stages []*mission.Stage

  for stageIdx := range plan.Stages {
    s := mission.Stage{
      Name:       plan.Stages[stageIdx].Name,
      Upstream:   plan.Stages[stageIdx].Upstream,
      Downstream: plan.Stages[stageIdx].Downstream,
    }

    stages = append(stages, &s)
    fmt.Println(s.Upstream, s.Downstream)
  }

  m := mission.New(plan.Name, stages)

  return &m
}
