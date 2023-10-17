
# Houston Go Client

Example usage:

```go
package main

import "github.com/datasparq-ai/houston/client"

func main() {

	// provide a key and base URL to initialise the client
	houston := client.New("my-houston-key", "https://houston.example.com")
	err := houston.SavePlan("./my_plan.json")

}
```

The client will use environment variables by default: 

```go
// this uses the 'HOUSTON_KEY' and 'HOUSTON_BASE_URL' environment variables for key and baseUrl respectively
houston := client.New("", "")
```

```go
houston := client.New("", "")

// we have previously saved a plan with this name
// a mission ID is not provided, so one will be created for us
res, err := houston.CreateMission("my-plan", "")

// trigger first stage
...
```

The same can be achieved with the 'start' command. See [commands](../docs/commands.md):

```go
houston := client.New("", "")
res, err := houston.Start("my-plan", "", nil, nil, nil)
```

It may be easier to start missions with the command line tool. The equivalent would be:

```bash
houston start --plan my-plan
```

Within services, the client is used to start and finish the stage:

```go
// get mission ID and stage name from trigger message
...

houston := client.New("", "")
res, err = houston.StartStage(missionId, stageName, false)

// complete task
...

res, err = houston.FinishStage(missionId, stageName, false)
for _, nextStage := range res.Next {
	// trigger stage
	...
}
```