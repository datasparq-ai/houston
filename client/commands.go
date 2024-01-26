package client

import "fmt"

// Start starts a new mission from the plan provided
func Start(plan string, id string, stages []string, exclude []string, skip []string, params map[string]interface{}) error {
	client := New("", "")
	mission, err := client.CreateMission(plan, id, params)
	if err != nil {
		return err
	}
	for _, s := range exclude {
		if s == "" {
			continue
		}
		_, err := client.ExcludeStage(mission.Id, s)
		if err != nil {
			return err
		}
	}
	for _, s := range skip {
		if s == "" {
			continue
		}
		_, err := client.SkipStage(mission.Id, s)
		if err != nil {
			return err
		}
	}
	// TODO: if stages is empty, determine starting stages
	//for _, s := range stages {
	// TODO: trigger stages (find service, determine trigger method)
	//client.Trigger(s)
	//}
	fmt.Println("New mission started with ID: " + mission.Id)
	return nil
}

func Save(plan string) error {
	client := New("", "")
	err := client.SavePlan(plan)
	if err != nil {
		return err
	}
	fmt.Println("Saved plan.")
	return nil
}

func CreateKey(id, name, password string) error {
	client := New("", "")
	key, err := client.CreateKey(id, name, password)
	if err != nil {
		return err
	}
	fmt.Printf(key)
	return nil
}

func ListKeys(password string) error {
	client := New("", "")
	keys, err := client.ListKeys(password)
	if err != nil {
		return err
	}
	for i := range keys {
		fmt.Println(keys[i])
	}
	return nil
}
