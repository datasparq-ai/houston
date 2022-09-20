package mission

import (
  "encoding/json"
  "fmt"
  "time"
)

// Response given when the user requests to change the state of a stage, e.g. start, finish, ignore, skip.
type Response struct {
  Success    bool     `json:"success"`
  Next       []string `json:"next"`
  IsComplete bool     `json:"complete"`
}

type PlanValidationError error

type Mission struct {
  Id         string            `json:"i" name:"id"`
  Name       string            `json:"n" name:"name"`     // the plan name, note: not needed in a mission
  Services   []string          `json:"a" name:"services"` // note: not needed in a mission but kept for convenience
  Stages     []*Stage          `json:"s" name:"stages"`
  Params     map[string]string `json:"p" name:"params"` // note: not needed in a mission but kept for convenience
  Start      time.Time         `json:"t" name:"start"`
  End        time.Time         `json:"e" name:"end"`
  isComplete bool
  graph      *Graph
}

// NewFromJSON creates missions objects from their database representation in JSON.
// Runs every time the mission is modified.
func NewFromJSON(jsonString []byte) (Mission, error) {
  var m Mission
  err := json.Unmarshal(jsonString, &m)
  if err != nil {
    return Mission{}, err
  }

  m.graph = NewGraph(&m)

  return m, err
}

// New creates a new mission from the minimum amount of information.
// Should only be used when converting plans to missions for validation.
func New(name string, stages []*Stage) Mission {
  m := Mission{Name: name, Stages: stages}
  m.graph = NewGraph(&m)
  return m
}

// Bytes converts mission to it's json representation as bytes. Used for outputting missions.
func (m *Mission) Bytes() []byte {

  output, err := json.Marshal(m)

  if err != nil {
    panic(err)
  }

  return output
}

// Validate tests mission is valid using the following logic:
// - more than 0 stages
// - no duplicate stage names
// - all referenced stages exist
// - graph is not cyclic
// - graph is contiguous (no orphaned stages)
// - [ ] stage parameters within length limit (TODO)
// - [ ] number of stages within limit (TODO)
func (m *Mission) Validate() PlanValidationError {

  // are there more than 0 stages?
  if len(m.Stages) == 0 {
    err := fmt.Errorf("invalid plan: must have more than 0 stages")
    return err
  }

  // are there any duplicate stage names?
  var stageNames []string
  for _, s := range m.Stages {
    if contains(stageNames, s.Name) {
      err := fmt.Errorf("invalid plan: stage name '%v' is not unique", s.Name)
      return err
    } else {
      stageNames = append(stageNames, s.Name)
    }
  }

  // are all stages referred to in upstream/downstream defined?
  for _, s := range m.Stages {
    for _, u := range s.Upstream {
      if !contains(stageNames, u) {
        err := fmt.Errorf("invalid plan: stage '%v' has upstream dependency '%v' which is not defined", s.Name, u)
        return err
      }
    }
    for _, d := range s.Downstream {
      if !contains(stageNames, d) {
        err := fmt.Errorf("invalid plan: stage '%v' has downstream dependency '%v' which is not defined", s.Name, d)
        return err
      }
    }
  }

  // is graph cyclic?
  // follow every path in the graph starting from each stage. If a stage ends up visiting itself then it's cyclic
  visited := make(map[*Stage]bool)
  recursion := make(map[*Stage]bool)
  for _, s := range m.Stages {
    visited[s] = false
    recursion[s] = false
  }
  for _, s := range m.Stages {
    if !visited[s] {
      if m.graph.CheckForCycle(s, visited, recursion) {
        err := fmt.Errorf("invalid plan: stage '%v' is dependent on itself (infinite loop)", s.Name)
        return err
      }
    }
  }

  // is graph contiguous?
  // follow every path forwards and backwards from a single node and check that every node was visited at least once
  if unreachableStage := m.graph.CheckForIncontiguity(m.Stages); unreachableStage != nil {
    err := fmt.Errorf("invalid plan: not contiguous - '%v' cannot be reached from '%v'", unreachableStage.Name, m.Stages[0].Name)
    return err
  }

  return nil
}

// Print prints all the information about the mission.
func (m *Mission) Print() {
  fmt.Print("Mission\nid:", m.Id, m.isComplete)
  if m.isComplete {
    fmt.Print("[complete]\n")
  }
  fmt.Println("name:", m.Name)
  fmt.Println("stages:")
  for _, s := range m.Stages {
    s.Print()
  }
}

// Report prints a text alternative to the mission dashboard.
func (m *Mission) Report() {
  fmt.Print(m.Name, "/", m.Id)
  if m.isComplete {
    fmt.Print(" [complete]")
  }
  fmt.Print("\n")
  for _, s := range m.Stages {
    fmt.Println(stateIcons[s.State], s.Name, s.PrintDuration())
  }
}

// exists so that one can find a stage within a stage list without needing a mission.
func getStage(stageName string, stages []*Stage) (*Stage, error) {
  for _, stage := range stages {
    if stage.Name == stageName {
      return stage, nil
    }
  }
  err := fmt.Errorf("no stage found with name '%v'", stageName)
  return nil, err
}

func (m *Mission) GetStage(stageName string) (*Stage, error) {
  s, err := getStage(stageName, m.Stages)
  return s, err
}

// CheckComplete marks the mission as complete if all stages have been finished, excluded, or skipped.
// Should run every time a stage is finished, excluded or skipped.
func (m *Mission) CheckComplete() {
  for _, s := range m.Stages {
    switch s.State {
    case finished, excluded, skipped:
      continue
    default:
      m.isComplete = false
      return
    }
  }
  m.isComplete = true
  m.End = time.Now()
}

// Next finds all stages that are eligible to run.
func (m *Mission) Next() []string {

  var nextStages []string

  for _, stage := range m.Stages {
    if stage.State != ready {
      continue
    }
    // all upstream stages must be finished or skipped
    if !m.graph.areUpstreamFinished(stage) {
      continue
    }
    nextStages = append(nextStages, stage.Name)
  }

  return nextStages
}

//
// below are methods that can be used directly by the API
//

// StartStage changes a stage's state to started using the following logic:
// - does stage exist?
// - is stage ready or failed (all other states are not allowed)?
// - are all upstream dependencies finished or skipped?
func (m *Mission) StartStage(stageName string, ignoreDependencies bool) (Response, error) {
  if m.isComplete {
    return Response{false, nil, true},
      fmt.Errorf("mission has been completed, cannot operate further")
  }
  s, err := m.GetStage(stageName)
  if err != nil {
    return Response{false, nil, m.isComplete}, err
  }

  // has stage already started or is it already finished?
  // has stage been excluded or skipped?
  switch s.State {
  case ready, failed:
    // ok, failed stages can be started again (retry)
  case started:
    err := fmt.Errorf("cannot start stage '%v' because it has already started - stages can only be started again after they have been marked as failed", stageName)
    return Response{false, nil, m.isComplete}, err
  case finished:
    err := fmt.Errorf("cannot start stage '%v' because it has already finished", stageName)
    return Response{false, nil, m.isComplete}, err
  case excluded:
    err := fmt.Errorf("cannot start stage '%v' because it is being excluded", stageName)
    return Response{false, nil, m.isComplete}, err
  case skipped:
    err := fmt.Errorf("cannot start stage '%v' because it was skipped", stageName)
    return Response{false, nil, m.isComplete}, err
  default:
    panic("unknown state")
  }

  if ignoreDependencies {
    // mark all upstream stages as excluded recursively so that stage can be started
    // pre-set this stage's state to excluded to prevent its downstream stages from being excluded by excludeUpstreamRecursively
    s.State = excluded
    err := m.excludeUpstreamRecursively(s)
    if err != nil {
      return Response{false, nil, m.isComplete}, err
    }
  }

  s.State = ready

  // are all upstream dependencies finished, excluded, or skipped?
  if !m.graph.areUpstreamFinished(s) {
    var err error
    // if not then find the unfinished stage in order to provide a helpful error message
    for _, dependency := range m.graph.up[s] {
      switch dependency.State {
      case finished, excluded:
        continue
      case skipped:
        if m.graph.areUpstreamFinished(s) {
          continue
        } else {
          err = fmt.Errorf("cannot start stage '%v' because skipped stage '%v' has unfinished upstream dependencies", stageName, dependency.Name)
        }
      case started, failed, ready:
        err = fmt.Errorf("cannot start stage '%v' because it has unfinished upstream dependency '%v'", stageName, dependency.Name)
      }
    }
    return Response{false, nil, m.isComplete}, err
  }

  // change the state
  s.State = started
  s.Start = time.Now()

  return Response{true, []string{}, m.isComplete}, nil
}

// FinishStage changes a stage's state to finished using the following logic:
// - does stage exist?
// - is stage ready or failed (all other states are not allowed)?
// - are all upstream dependencies finished or skipped?
func (m *Mission) FinishStage(stageName string, ignoreDependencies bool) (Response, error) {
  if m.isComplete {
    return Response{false, nil, true},
      fmt.Errorf("mission has been completed, cannot operate further")
  }
  s, err := m.GetStage(stageName)
  if err != nil {
    return Response{false, nil, m.isComplete}, err
  }

  // has stage already finished or is it already finished?
  // has stage been excluded or skipped?
  switch s.State {
  case started:
    // ok
  case excluded, skipped, ready:
    err := fmt.Errorf("cannot finish stage '%v' because it has not been started", stageName)
    return Response{false, nil, m.isComplete}, err
  case finished:
    err := fmt.Errorf("stage '%v' is already finished", stageName)
    return Response{false, nil, m.isComplete}, err
  case failed:
    err := fmt.Errorf("cannot finish stage '%v' because it is marked as failed", stageName)
    return Response{false, nil, m.isComplete}, err
  default:
    panic("unknown state") // should be impossible
  }

  // change the state
  s.State = finished
  s.End = time.Now()

  if ignoreDependencies {
    // mark all downstream stages as excluded so that they don't run next
    err := m.excludeDownstreamRecursively(s)
    if err != nil {
      return Response{false, nil, m.isComplete}, err
    }
  }

  // find the next stages/check if mission is finished
  nextStages := m.Next()
  if len(nextStages) == 0 {
    m.CheckComplete()
  }

  return Response{true, nextStages, m.isComplete}, nil
}

// SkipStage changes a stage's state to skip using the following logic:
// - does stage exist?
// - state can't be started, failed, excluded or already skipped?
// - are all upstream dependencies finished or skipped?
func (m *Mission) SkipStage(stageName string) (Response, error) {
  if m.isComplete {
    return Response{false, nil, true},
      fmt.Errorf("mission has been completed, cannot operate further")
  }
  // Does stage exist
  s, err := m.GetStage(stageName)
  if err != nil {
    return Response{false, nil, m.isComplete}, err
  }

  //   Check the state of the stage
  switch s.State {
  case ready:
    s.State = skipped
  case skipped, excluded, finished:
    // this is allowed, but state will not be changed - mission logic should not be affected
  case started, failed:
    err := fmt.Errorf("cannot skip stage '%v' because it has previously been %s", stageName, s.State)
    return Response{false, nil, m.isComplete}, err
  default:
    panic("unknown state") // should be impossible
  }

  nextStages := m.Next()
  if len(nextStages) == 0 {
    m.CheckComplete()
  }

  return Response{true, nextStages, m.isComplete}, nil
}

// FailStage changes a stage's state to failed using the following logic:
// - does stage exist?
// - state can't be ready, failed, excluded, skipped or failed, just started
func (m *Mission) FailStage(stageName string) (Response, error) {
  if m.isComplete {
    return Response{false, nil, true},
      fmt.Errorf("mission has been completed, cannot operate further")
  }
  // does stage exist
  s, err := m.GetStage(stageName)
  if err != nil {
    return Response{false, nil, m.isComplete}, err
  }

  // check the state of the stage
  switch s.State {
  case started:
    // ok
  case ready, excluded, skipped, finished, failed:
    err := fmt.Errorf("cannot fail stage '%v' because it is %s, not started", stageName, s.State)
    return Response{false, nil, m.isComplete}, err
  default:
    panic("unknown state") // should be impossible
  }

  s.State = failed
  return Response{true, []string{}, false}, nil
}

// ExcludeStage changes a stage's state to excluded using the following logic:
// - does stage exist?
// - state can't be started, finished, failed or skipped
// - are all upstream dependencies ready?
// - all downstream dependencies must be excluded too
func (m *Mission) ExcludeStage(stageName string) (Response, error) {
  if m.isComplete {
    return Response{false, nil, true},
      fmt.Errorf("mission has been completed, cannot operate further")
  }
  // does stage exist
  s, err := m.GetStage(stageName)

  if err != nil {
    return Response{false, nil, m.isComplete}, err
  }

  err = m.tryExcludingStage(s)
  if err != nil {
    return Response{false, nil, m.isComplete}, err
  }

  // next exclude all downstream recursively
  err = m.excludeDownstreamRecursively(s)
  if err != nil {
    return Response{false, nil, m.isComplete}, err
  }

  m.CheckComplete()
  return Response{true, []string{}, m.isComplete}, nil
}

func (m *Mission) tryExcludingStage(s *Stage) error {
  switch s.State {
  case ready, excluded:
    s.State = excluded
    return nil
  case finished, skipped:
    // this is allowed, but state will not be changed - mission logic should not be affected
    return nil
  case started, failed:
    err := fmt.Errorf("cannot exclude stage '%v' because it is %s, not ready", s.Name, s.State)
    return err
  }
  return nil
}

func (m *Mission) excludeDownstreamRecursively(s *Stage) error {
  for _, downstreamStage := range m.graph.down[s] {
    if downstreamStage.State == excluded {
      // already seen, don't recurse
    } else {
      err := m.tryExcludingStage(downstreamStage)
      if err != nil {
        return err
      }
      err = m.excludeDownstreamRecursively(downstreamStage)
      if err != nil {
        return err
      }
    }
  }
  return nil
}

// excludeUpstreamRecursively is only run when StartStage is run with ignoreDependencies set to true. It is used
// to ensure that the stage can start without its dependencies being finished by excluding then all and also excludes
// any stages that can no longer run due to their dependencies being excluded
func (m *Mission) excludeUpstreamRecursively(s *Stage) error {
  for _, upstreamStage := range m.graph.up[s] {
    if upstreamStage.State == excluded {
      // already seen, don't recurse
    } else {
      err := m.tryExcludingStage(upstreamStage)
      if err != nil {
        return err
      }
      err = m.excludeUpstreamRecursively(upstreamStage)
      if err != nil {
        return err
      }
      // now exclude any stages that can no longer run because their dependencies are excluded
      err = m.excludeDownstreamRecursively(upstreamStage)
      if err != nil {
        return err
      }
    }
  }
  return nil
}
