package model

import "github.com/datasparq-ai/houston/mission"

type Error struct {
  Message string `json:"message"`
}

type Success struct {
  Message string `json:"message"`
}

type Key struct {
  Id    string `json:"id"`
  Name  string `json:"name" key:"n"`
  Usage string `json:"usage" key:"u"`
}

// Mission is returned by GET /missions/{id} and should only be used for checking mission status
type Mission mission.Mission

type MissionStage mission.Stage

type MissionStageStateUpdateResponse mission.Response

type MissionCreateRequest struct {
  Plan   string                 `json:"plan"`
  Id     string                 `json:"id"`
  Params map[string]interface{} `json:"params"` // TODO: update plan params with mission params
}

type MissionCreatedResponse struct {
  Id string `json:"id"`
}

type MissionStageStateUpdate struct {
  State              string `json:"state"`
  IgnoreDependencies bool   `json:"ignoreDependencies"`
}

type Stage struct {
  Name       string                 `json:"name" key:"n"`
  Service    string                 `json:"service" key:"a"`
  Upstream   []string               `json:"upstream" key:"u"`
  Downstream []string               `json:"downstream" key:"d"`
  Params     map[string]interface{} `json:"params" key:"p"`
}

type Plan struct {
  Name     string                 `json:"name" key:"n"`
  Services []string               `json:"services" key:"a"`
  Stages   []*Stage               `json:"stages" key:"s"`
  Params   map[string]interface{} `json:"params" key:"p"`
}
