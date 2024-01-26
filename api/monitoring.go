package api

import (
	"encoding/json"
	"github.com/datasparq-ai/houston/database"
	"github.com/datasparq-ai/houston/mission"
	"runtime"
	"time"
)

// Monitor checks the health of the API server and performs other duties at regular intervals
func (a *API) Monitor() {
	for {
		a.DeleteExpiredMissions()
		a.HealthCheck()
		time.Sleep(12 * time.Hour)
	}
}

func (a *API) HealthCheck() {
	log.Info("Checking the health of the database")
	err := a.db.Health()
	if err != nil {
		switch err.(type) {
		case *database.MemoryUsageError:
			log.Error(err.Error())
		default:
			log.Error(err.Error())
		}
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryLimitBytes := a.config.MemoryLimitMiB * 1024 * 1024

	log.Infof("Health check: total memory usage is %v bytes", m.Alloc)
	if int64(m.Alloc) > memoryLimitBytes {
		log.Warnf("Houston is using more memory than the safe limit; %v out of %v bytes used.", m.Alloc, memoryLimitBytes)
	}

}

// DeleteExpiredMissions looks for missions in the 'complete' and 'active' lists that are older than the
// Config.MissionExpiry time and deletes them
func (a *API) DeleteExpiredMissions() {

	keys, err := a.db.ListKeys()
	if err != nil {
		log.Error(err)
		return
	}
	log.Infof("Found '%v' keys\n", len(keys))

	for _, key := range keys {
		deletedMissions := 0

		log.Infof("Looking at key '%v' for completed missions\n", key)
		missions := a.CompletedMissions(key)
		log.Infof("Found %v completed missions\n", len(missions))

		for _, missionId := range missions {
			log.Infof("Found completed mission '%v'\n", missionId)

			if a.missionCanBeDeleted(key, missionId) {
				a.DeleteMission(key, missionId)
				deletedMissions++
				log.Infof("Deleted '%v'\n", missionId)
			} else {
				log.Infof("Not deleting mission because it's not old enough")
				// completed missions are stored in chronological order, so we can stop the loop now
				break
			}
		}

		log.Infof("Looking at key '%v' for active but expired missions\n", key)

		plans, err := a.ListPlans(key)
		if err != nil {
			log.Error(err)
			continue
		}

		for _, planName := range plans {
			missions = a.ActiveMissions(key, planName)
			log.Debugf("Found %v active missions for plan '%v'\n", len(missions), planName)

			for _, missionId := range missions {
				log.Debugf("Found active mission '%v'\n", missionId)
				if a.missionCanBeDeleted(key, missionId) {
					a.DeleteMission(key, missionId)
					deletedMissions++
					log.Infof("Deleted '%v'\n", missionId)
				} else {
					log.Infof("Not deleting mission because it's still in use")
					// active missions are stored in chronological order, so we can stop the loop now
					break
				}
			}
		}
		log.Infof("Deleted %v missions from key '%v'\n", deletedMissions, key)
	}
}

// missionCanBeDeleted returns true if the server configuration allows for a mission to be deleted by the API monitor
func (a *API) missionCanBeDeleted(key string, missionId string) bool {
	missionString, ok := a.db.Get(key, missionId)
	var miss mission.Mission
	// if mission can't be read from db, then delete it anyway
	if !ok {
		log.Debugf("Mission '%v' will be deleted because can't read from the database\n", missionId)
		return true
	} else {
		err := json.Unmarshal([]byte(missionString), &miss)
		if err != nil {
			log.Debugf("Mission '%v' will be deleted because can't be parsed as JSON, which makes it invalid\n", missionId)
			return true
		} else {
			if miss.Start.IsZero() {
				log.Debugf("Mission '%v' will be deleted because it has no start time, which makes it invalid\n", missionId)
				return true
			} else {
				if !miss.End.IsZero() && miss.End.Before(time.Now().Add(a.config.MissionExpiry)) {
					log.Debugf("Mission '%v' will be deleted because it ended over %s ago\n", missionId, a.config.MissionExpiry)
					return true
				} else if miss.Start.Before(time.Now().Add(a.config.MissionExpiry)) {
					log.Debugf("Mission '%v' will be deleted because it started over %s ago\n", missionId, a.config.MissionExpiry)
					return true
				}
			}
		}
	}
	log.Debugf("Mission '%v' won't be deleted because it started under %s ago\n", missionId, a.config.MissionExpiry)
	return false
}
