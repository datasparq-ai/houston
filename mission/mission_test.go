/**
This file tests mission logic only. There is no database used (everything is in memory).
The following capabilities are covered by these tests:
- creating missions
- not creating invalid/incorrectly formatted missions
- not creating cyclic or incontiguous missions
- changing stage state when allowed (e.g. started -> finished) for all types of state
- failing to change stage state when not allowed (e.g. started -> started) for all types of state
- completing missions with skipped stages
- ignoring upstream/downstream dependencies
- completing missions with excluded stages
- not completing when stages aren't finished, skipped, or excluded
- not being able to run a mission out of order

to run:

    go test -v ./mission

*/
package mission

import (
  "io/ioutil"
  "testing"
)

func TestMission_StartStage_ErrorAlreadyStarted(t *testing.T) {

  // create new mission from plan
  data, _ := ioutil.ReadFile("../tests/test_mission.json")
  m, _ := NewFromJSON(data)
  err := m.Validate()
  if err != nil {
    t.Fatalf(`Test mission didn't pass validation.`)
  }

  m.StartStage("stage-1", false)
  _, err = m.StartStage("stage-1", false)

  if err == nil {
    t.Fatalf(`Should result in an error because stage was already started`)
  }
}

func TestMission_StartStage_IgnoreDependencies(t *testing.T) {

  // create new mission from plan
  data, _ := ioutil.ReadFile("../tests/test_mission.json")
  m, _ := NewFromJSON(data)
  m.Validate()

  _, err := m.StartStage("stage-2", true)
  if err != nil {
    t.Fatalf(`Stage should be able to start if dependencies are ignored`)
  }
  m.FinishStage("stage-2", false)
  s, _ := m.GetStage("stage-1")
  if s.State != excluded {
    t.Fatalf(`First stage should be excluded`)
  }
  _, err = m.StartStage("stage-1", false)
  if err == nil {
    t.Fatalf(`First stage should not be able to start due to being excluded`)
  }
  if !m.isComplete {
    t.Fatalf(`Mission should be complete`)
  }
}

func TestMission_FinishStage_IgnoreDependencies(t *testing.T) {

  // create new mission from plan
  data, _ := ioutil.ReadFile("../tests/test_mission.json")
  m, _ := NewFromJSON(data)
  m.Validate()

  m.StartStage("stage-1", false)
  m.FinishStage("stage-1", true)
  s, _ := m.GetStage("stage-2")
  if s.State != excluded {
    t.Fatalf(`Second stage should be excluded`)
  }
  _, err := m.StartStage("stage-2", false)
  if err == nil {
    t.Fatalf(`Second stage should not be able to start due to being excluded`)
  }
  if !m.isComplete {
    t.Fatalf(`Mission should be complete`)
  }
}

func TestMission_SkipStage(t *testing.T) {
  data, _ := ioutil.ReadFile("../tests/test_mission.json")
  m, _ := NewFromJSON(data)
  err := m.Validate()
  if err != nil {
    t.Fatalf(`Test mission didn't pass validation.`)
  }

  m.StartStage("stage-1", false)
  m.FinishStage("stage-1", false)
  res, err := m.SkipStage("stage-2")
  if err != nil || !res.Success {
    t.Fatalf("Failed to skip stage.")
  }
  if !res.IsComplete {
    t.Fatalf("Mission should be complete.")
  }
  s, _ := m.GetStage("stage-2")
  if s.State != skipped {
    t.Fatalf("Stage state should be skipped.")
  }
}

func TestMission_SkipStage_Error(t *testing.T) {
  data, _ := ioutil.ReadFile("../tests/test_mission.json")
  m, _ := NewFromJSON(data)
  err := m.Validate()
  if err != nil {
    t.Fatalf(`Test mission didn't pass validation.`)
  }

  // attempt to start then skip the same stage - should get an error
  m.StartStage("stage-1", false)
  _, err = m.SkipStage("stage-1")
  if err == nil {
    t.Fatalf("Skip stage should result in an error.")
  }
  s, _ := m.GetStage("stage-1")
  if s.State != started {
    t.Fatalf("Stage state should be started.")
  }

  // attempt to finish then skip the same stage - should still be finished
  m.FinishStage("stage-1", false)
  _, _ = m.SkipStage("stage-1")
  s, _ = m.GetStage("stage-1")
  if s.State != finished {
    t.Fatalf("Stage state should be finished.")
  }
}

func TestMission_FailStage_Error(t *testing.T) {
  data, _ := ioutil.ReadFile("../tests/test_mission.json")
  m, _ := NewFromJSON(data)
  err := m.Validate()
  if err != nil {
    t.Fatalf(`Test mission didn't pass validation.`)
  }

  // test a stage that has not started can't fail
  _, err = m.FailStage("stage-1")
  if err == nil {
    t.Fatalf("Fail stage without starting should result in an error.")
  }

  // test a stage that has finished can't fail
  m.StartStage("stage-1", false)
  m.FinishStage("stage-1", false)
  _, err = m.FailStage("stage-1")
  if err == nil {
    t.Fatalf("Fail stage thats finished should result in an error.")
  }

}

func TestMission_FailStage(t *testing.T) {
  data, _ := ioutil.ReadFile("../tests/test_mission.json")
  m, _ := NewFromJSON(data)
  err := m.Validate()
  if err != nil {
    t.Fatalf(`Test mission didn't pass validation.`)
  }

  //   test a stage that has started can fail
  m.StartStage("stage-1", false)
  _, err = m.FailStage("stage-1")
  if err != nil {
    t.Fatalf("Started stage should fail")
  }

  // test a stage can't start after the previous stage has failed
  _, err = m.StartStage("stage-2", false)
  if err == nil {
    t.Fatalf("Started stage should fail")
  }
}

func TestMission_ExcludeStage(t *testing.T) {
  data, _ := ioutil.ReadFile("../tests/test_mission.json")
  m, _ := NewFromJSON(data)
  err := m.Validate()
  if err != nil {
    t.Fatalf(`Test mission didn't pass validation.`)
  }

  // test a stage that is ready can be excluded
  _, err = m.ExcludeStage("stage-1")
  s, _ := m.GetStage("stage-1")
  if err != nil || s.State != excluded {
    t.Fatalf("First stage should be excluded")
  }

  // test a stage can't start after it has been excluded
  _, err = m.StartStage("stage-1", false)
  if err == nil {
    t.Fatalf("Excluded stage should fail to start")
  }

  // test that stages that depend on the excluded stage are also excluded
  s, _ = m.GetStage("stage-2")
  if s.State != excluded {
    t.Fatalf("Downstream stage should be excluded after excluding first stage")
  }

  if !m.isComplete {
    t.Fatalf("Mission should be complete if all stages are excluded")
  }
}

func TestMission_ExcludeUpstream(t *testing.T) {
  data, _ := ioutil.ReadFile("../tests/test_mission_complex.json")
  m, _ := NewFromJSON(data)
  err := m.Validate()
  if err != nil {
    t.Fatalf(`Test mission didn't pass validation.`)
  }

  // all stages are either dependencies or dependants of dependencies of stage-2-a
  _, err = m.StartStage("stage-2a", true)
  _, err = m.FinishStage("stage-2a", true)
  if err != nil {
    t.Fatalf(`stage should start and finish without errors.`)
  }

  // test that stages that depend on stage-2-a's dependencies are also excluded
  s, _ := m.GetStage("stage-2b")
  if s.State != excluded {
    t.Fatalf("Stages that are dependent on excluded dependencies of the started stage should have " +
      "been marked as ignored by Mission.excludeUpstreamRecursively")
  }

  if !m.isComplete {
    t.Fatalf("Mission should be complete due to all stages being finished or excluded")
  }
}

func TestMission_Validate_CheckNotCyclic(t *testing.T) {
  data, _ := ioutil.ReadFile("../tests/test_mission_cyclic.json")
  m, _ := NewFromJSON(data)
  err := m.Validate()
  if err == nil {
    t.Fatalf(`Cyclic mission should not pass validation.`)
  }
}

func TestMission_Validate_CheckContiguous(t *testing.T) {
  data, _ := ioutil.ReadFile("../tests/test_mission_incontiguous.json")
  m, _ := NewFromJSON(data)
  err := m.Validate()
  if err == nil {
    t.Fatalf(`Incontiguous mission should not pass validation.`)
  }
}

// a mission with no links may cause unexpected behaviour as the graph will be completely empty
// TODO: test that mission with no links fails validation with correct error message (incontiguous)