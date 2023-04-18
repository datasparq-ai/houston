package main

import (
	"encoding/json"
	"fmt"
	"github.com/datasparq-ai/houston/database"
	"github.com/datasparq-ai/houston/mission"
	"time"
)

// Monitor checks the health of the API server and performs other duties at regular intervals
func (a *API) Monitor() {
	for {
		a.DeleteExpiredMissions()
		a.HealthCheck()
		time.Sleep(5 * time.Second)
		//time.Sleep(12 * time.Hour)
	}
}

func (a *API) HealthCheck() {
	fmt.Println("🩺 Checking the health of the database")
	err := a.db.Health()
	if err != nil {
		switch err.(type) {
		case *database.MemoryUsageError:
			fmt.Println("⚠️  " + err.Error()) // TODO: warning (and notification?)
		default:
			fmt.Println("⚠️  " + err.Error())
		}
	}
}

// DeleteExpiredMissions looks for missions in the 'complete' and 'active' lists that are older than the
// Config.MissionExpiry time and deletes them
func (a *API) DeleteExpiredMissions() {

	keys, err := a.db.ListKeys()
	if err != nil {
		//log.Error(err)
		fmt.Println(err)
		return
	}
	fmt.Printf("Found '%v' keys\n", len(keys))

	for _, key := range keys {
		deletedMissions := 0

		fmt.Printf("Looking at key '%v' for completed missions\n", key)
		missions := a.CompletedMissions(key)
		fmt.Printf("Found %v completed missions\n", len(missions))

		for _, missionId := range missions {
			fmt.Printf("Found completed mission '%v'\n", missionId)

			if a.missionCanBeDeleted(key, missionId) {
				a.deleteMission(key, missionId)
				deletedMissions++
				fmt.Printf("Deleted '%v'\n", missionId)
			} else {
				fmt.Println("Not deleting mission because it's not old enough")
				// completed missions are stored in chronological order, so we can stop the loop now
				break
			}
		}

		fmt.Printf("Looking at key '%v' for active but expired missions\n", key)

		plans, err := a.ListPlans(key)
		if err != nil {
			//log.Error(err)
			fmt.Println(err)
			continue
		}

		for _, planName := range plans {
			fmt.Printf("Found plan '%v'\n", planName)

			missions = a.ActiveMissions(key, planName)
			fmt.Printf("Found %v active missions\n", len(missions))

			for _, missionId := range missions {
				fmt.Printf("Found active mission '%v'\n", missionId)
				if a.missionCanBeDeleted(key, missionId) {
					a.deleteMission(key, missionId)
					deletedMissions++
					fmt.Printf("Deleted '%v'\n", missionId)
				} else {
					fmt.Println("Not deleting mission because it's still in use")
					// active missions are stored in chronological order, so we can stop the loop now
					break
				}
			}
		}
		fmt.Printf("Deleted %v missions from key '%v'\n", deletedMissions, key)
	}
}

// missionCanBeDeleted returns true if the server configuration allows for a mission to be deleted by the API monitor
func (a *API) missionCanBeDeleted(key string, missionId string) bool {
	missionString, ok := a.db.Get(key, missionId)
	var miss mission.Mission
	// if mission can't be read from db, then delete it anyway
	if !ok {
		fmt.Printf("Mission '%v' will be deleted because can't read from the database\n", missionId)
		return true
	} else {
		err := json.Unmarshal([]byte(missionString), &miss)
		if err != nil {
			fmt.Printf("Mission '%v' will be deleted because can't be parsed as JSON, which makes it invalid\n", missionId)
			return true
		} else {
			if miss.Start.IsZero() {
				fmt.Printf("Mission '%v' will be deleted because it has no start time, which makes it invalid\n", missionId)
				return true
			} else {
				if !miss.End.IsZero() && miss.End.Before(time.Now().Add(a.config.MissionExpiry)) {
					fmt.Printf("Mission '%v' will be deleted because it ended over %s ago\n", missionId, a.config.MissionExpiry)
					return true
				} else if miss.Start.Before(time.Now().Add(a.config.MissionExpiry)) {
					fmt.Printf("Mission '%v' will be deleted because it started over %s ago\n", missionId, a.config.MissionExpiry)
					return true
				}
			}
		}
	}
	fmt.Printf("Mission '%v' won't be deleted because it started over %s ago\n", missionId, a.config.MissionExpiry)
	return false
}
