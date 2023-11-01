package main

import (
	"encoding/json"
	"fmt"
	"github.com/datasparq-ai/houston/api"
	"github.com/datasparq-ai/houston/model"
	"github.com/spf13/cobra"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// demo creates a new API instance and runs demonstration missions
func demo(createCmd *cobra.Command) {
	configPath, _ := createCmd.Flags().GetString("config")

	s := "\u001B[1;38;2;58;145;172m"
	e := "\u001B[0m"
	fmt.Println("\u001B[1mHOUSTON DEMONSTRATION MODE\u001B[0m")
	fmt.Printf("\u001B[37mstarting a local API server%[2]v\n", s, e)
	fmt.Printf(">>> %[1]vhouston api%[2]v\n", s, e)

	a := api.New(configPath)
	go a.Run()
	go a.Monitor()

	time.Sleep(500 * time.Millisecond)

	fmt.Printf("\u001B[37mcreating an API key%[2]v\n", s, e)
	fmt.Printf(">>> %[1]vhouston create-key%[2]v \u001B[1m-i demo -n Demo%[2]v\n", s, e)
	fmt.Printf(">>> %[1]vexport%[2]v \u001B[1mHOUSTON_KEY=demo%[2]v\n", s, e)

	a.CreateKey("demo", "Demo")
	fmt.Println("Created key 'demo'")

	// https://history.nasa.gov/SP-4029/Apollo_11i_Timeline.htm
	planBytes := []byte(`{
  "name": "apollo",
  "stages": [
    {"name": "engine-ignition", "params": {"foo": "bar", "engines": 4, "bad": {"foo": "bar"}}},
    {"name": "engine-thrust-ok", "upstream": ["engine-ignition"]},
    {"name": "release-holddown-arms", "upstream": ["engine-thrust-ok"]},
    {"name": "umbilical-disconnected"},
    {"name": "liftoff", "upstream": ["release-holddown-arms", "umbilical-disconnected"]},
    {"name": "tower-clearance-yaw-maneuver", "upstream": ["liftoff"]},
    {"name": "pitch-and-roll-maneuver", "upstream": ["tower-clearance-yaw-maneuver"]},
    {"name": "apex", "upstream": ["pitch-and-roll-maneuver"]},
    {"name": "mach-one", "upstream": ["liftoff"]},
    {"name": "outboard-engine-cutoff", "upstream": ["mach-one"]},
    {"name": "iterative-guidance-mode", "upstream": ["pitch-and-roll-maneuver"]}
  ]
}`)
	var plan model.Plan
	err := json.Unmarshal(planBytes, &plan)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\u001B[37msaving new plan to the API database%[2]v\n", s, e)
	fmt.Printf(">>> %[1]vcat%[2]v \u001B[1mpath/to/plan.json%[2]v\n", s, e)
	fmt.Println(string(planBytes))
	fmt.Printf(">>> %[1]vhouston save%[2]v \u001B[1m-p path/to/plan.json%[2]v\n", s, e)
	err = a.SavePlan("demo", plan)

	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	fmt.Println("Created plan 'apollo'")

	fmt.Printf("\u001B[37mcreating a new mission%[2]v\n", s, e)
	fmt.Printf(">>> %[1]vhouston start%[2]v \u001B[1m-p apollo -i apollo-11%[2]v\n", s, e)

	_, err = a.CreateMissionFromPlan("demo", "apollo", "apollo-11")
	if err != nil {
		panic(err)
	}

	fmt.Println("Created mission with ID 'apollo-11'")

	fmt.Printf("\u001B[37mgo to http://localhost:8000?key=demo to view this mission on the dashboard using the key 'demo'%[2]v\n", s, e)

	time.Sleep(2 * time.Second)
	a.UpdateStageState("demo", "apollo-11", "engine-ignition", "started", false)
	time.Sleep(1234 * time.Millisecond)
	res, _ := a.UpdateStageState("demo", "apollo-11", "engine-ignition", "finished", false)
	go continueMission(a, "demo", "apollo-11", res.Next)

	time.Sleep(3 * time.Second)

	go func() {

		missionCount := 1
		for missionCount < 30 {
			time.Sleep(time.Duration(rand.Float32()*5000.0) * time.Millisecond)
			//fmt.Printf(">>> %[1]vhouston start%[2]v \u001B[1m-p apollo -i apollo-12%[2]v\n", s, e)
			missionId := fmt.Sprintf("apollo-%v", 11+missionCount)
			_, err = a.CreateMissionFromPlan("demo", "apollo", missionId)
			if err != nil {
				panic(err)
			}

			time.Sleep(time.Duration(rand.Float32()*1000.0) * time.Millisecond)
			a.UpdateStageState("demo", missionId, "engine-ignition", "started", false)
			res, _ = a.UpdateStageState("demo", missionId, "engine-ignition", "finished", false)
			go continueMission(a, "demo", missionId, res.Next)

			missionCount++
		}
	}()

	// keep the API running until the user exits
	exitSignal := make(chan os.Signal)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal

}

// continueMission completes every stage of a mission recursively, for demo purposes
func continueMission(a *api.API, key, missionId string, stages []string) {
	for _, stageName := range stages {
		time.Sleep(time.Millisecond * time.Duration(rand.Float32()*500.0))
		a.UpdateStageState("demo", missionId, stageName, "started", false)
		time.Sleep(time.Millisecond * time.Duration(rand.Float32()*10000.0))
		res, _ := a.UpdateStageState("demo", missionId, stageName, "finished", false)
		go continueMission(a, key, missionId, res.Next)
	}
}
