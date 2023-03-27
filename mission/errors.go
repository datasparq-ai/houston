package mission

import "fmt"

type PlanValidationError struct {
	Detail string
}

func (e *PlanValidationError) Error() string {
	return "plan is invalid: " + e.Detail
}

type StageChangeError struct {
	Detail string
}

func (e *StageChangeError) Error() string {
	return "invalid state change: " + e.Detail
}

type CompletedError struct{}

func (e *CompletedError) Error() string {
	return "mission has been completed, cannot operate further"
}

type StageNotFoundError struct {
	StageName string
}

func (e *StageNotFoundError) Error() string {
	return fmt.Sprintf("no stage found with name '%v'", e.StageName)
}
