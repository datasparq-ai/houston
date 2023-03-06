package mission

import (
	"fmt"
	"time"
)

type Stage struct {
	Name       string                 `json:"n" name:"name"`
	Service    string                 `json:"a" name:"service"`
	Upstream   []string               `json:"u" name:"upstream"`
	Downstream []string               `json:"d" name:"downstream"`
	Params     map[string]interface{} `json:"p" name:"params"`
	State      state                  `json:"s" name:"state"`
	Start      time.Time              `json:"t" name:"start"`
	End        time.Time              `json:"e" name:"end"`
}

type state int

const (
	ready state = iota
	started
	finished
	failed
	excluded
	skipped
)

func (s state) String() string {
	//allowing the stage names to be written back

	states := [...]string{
		"ready",
		"started",
		"finished",
		"failed",
		"excluded",
		"skipped"}

	if s < ready || s > skipped {
		return "Unknown state"
	}
	return states[s]
}

var stateIcons = []string{"○", "◎", "◍", "!", "x", "-"}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
func stageListContains(s []*Stage, e *Stage) bool {
	for _, a := range s {
		if a.Name == e.Name {
			return true
		}
	}
	return false
}

func (s *Stage) Print() {
	fmt.Println(" - name:", s.Name)
	fmt.Println("   params:")
	for k, v := range s.Params {
		fmt.Println("    ", k, v)
	}
}

func (s *Stage) PrintDuration() string {
	if s.Start.IsZero() {
		return "-"
	} else {
		if s.End.IsZero() {
			return time.Now().Sub(s.Start).String()
		} else {
			return s.End.Sub(s.Start).String()
		}
	}
}
