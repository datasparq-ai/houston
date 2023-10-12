package main

import (
	"encoding/json"
	"fmt"
	"github.com/datasparq-ai/houston/model"
	"github.com/spf13/cobra"
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

	api := New(configPath)
	go api.Run()
	go api.Monitor()

	time.Sleep(500 * time.Millisecond)

	fmt.Printf("\u001B[37mcreating an API key%[2]v\n", s, e)
	fmt.Printf(">>> %[1]vhouston create-key%[2]v \u001B[1m-i demo -n Demo%[2]v\n", s, e)
	fmt.Printf(">>> %[1]vexport%[2]v \u001B[1mHOUSTON_KEY=demo%[2]v\n", s, e)

	api.CreateKey("demo", "Demo")
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
    {"name": "tower-clearance-yaw-maneuver", "upstream": ["liftoff"]}
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
	err = api.SavePlan("demo", plan)

	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	fmt.Println("Created plan 'apollo'")

	fmt.Printf("\u001B[37mcreating a new mission%[2]v\n", s, e)
	fmt.Printf(">>> %[1]vhouston start%[2]v \u001B[1m-p apollo -i apollo-11%[2]v\n", s, e)

	_, err = api.CreateMissionFromPlan("demo", "apollo", "apollo-11")
	if err != nil {
		panic(err)
	}

	fmt.Println("Created mission with ID 'apollo-11'")

	fmt.Printf("\u001B[37mgo to http://localhost:%[3]v to view this mission on the dashboard using the key 'demo'%[2]v\n", s, e, api.config.Port)

	time.Sleep(2 * time.Second)
	api.UpdateStageState("demo", "apollo-11", "engine-ignition", "started", false)
	time.Sleep(1342 * time.Millisecond)
	api.UpdateStageState("demo", "apollo-11", "engine-ignition", "finished", false)
	api.UpdateStageState("demo", "apollo-11", "engine-thrust-ok", "started", false)
	time.Sleep(1042 * time.Millisecond)
	api.UpdateStageState("demo", "apollo-11", "engine-thrust-ok", "finished", false)
	api.UpdateStageState("demo", "apollo-11", "umbilical-disconnected", "started", false)
	api.UpdateStageState("demo", "apollo-11", "release-holddown-arms", "started", false)
	time.Sleep(1820 * time.Millisecond)
	api.UpdateStageState("demo", "apollo-11", "umbilical-disconnected", "finished", false)
	time.Sleep(1820 * time.Millisecond)
	api.UpdateStageState("demo", "apollo-11", "release-holddown-arms", "finished", false)
	time.Sleep(1120 * time.Millisecond)
	api.UpdateStageState("demo", "apollo-11", "liftoff", "started", false)
	time.Sleep(1560 * time.Millisecond)
	api.UpdateStageState("demo", "apollo-11", "liftoff", "finished", false)
	api.UpdateStageState("demo", "apollo-11", "tower-clearance-yaw-maneuver", "started", false)
	time.Sleep(3440 * time.Millisecond)
	api.UpdateStageState("demo", "apollo-11", "tower-clearance-yaw-maneuver", "finished", false)

	fmt.Printf(">>> %[1]vhouston start%[2]v \u001B[1m-p apollo -i apollo-12%[2]v\n", s, e)
	_, err = api.CreateMissionFromPlan("demo", "apollo", "apollo-12")
	if err != nil {
		panic(err)
	}

	fmt.Println("Created mission with ID 'apollo-12'")

	time.Sleep(1 * time.Second)
	api.UpdateStageState("demo", "apollo-12", "engine-ignition", "started", false)
	time.Sleep(1342 * time.Millisecond)

	// keep the API running until the user exits
	exitSignal := make(chan os.Signal)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM)
	<-exitSignal

}
