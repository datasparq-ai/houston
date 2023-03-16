
## Commands

Commands are high-level methods that allow users or Houston integrated services to carry out common tasks
with a single command (e.g. starting a mission), avoiding the need for any additional scripts. 

A Houston integrated service will run a command when triggered with a message containing the
'command' attribute. The service will automatically find the API key and initialise a Houston client.

The API key must be made available as the `HOUSTON_KEY` environment variable, 
or through one of the methods defined for the cloud provider/plugin being used, see:
- [Providing the API Key: Google Cloud Platform](./google_cloud.md#providing-the-api-key)  

### Save

Save a plan or update an existing plan.

Example CLI command:

```bash
houston save --plan=gs://my-bucket/apollo.yaml
```

Example Python script - this requires a Houston API key to be available either as environment variable or using an alternative method:

```python
from houston import save

plan = {
    "name": "apollo",
    "stages": [
        {
            "name": "foo"
        },
        {
            "name": "bar",
            "upstream": "foo"
        }
    ]
}

save(plan=plan)
```

Example message to GCP Houston service:

```json
{
  "command": "save",
  "plan":  "gs://my-bucket/apollo.yaml"
}
```

Note that the service will need `houston-client[gcp]` installed to be able to retrieve the plan from Google Cloud Storage. 

### Delete

Delete a plan.

Example CLI command:

```bash
houston delete --plan=gs://my-bucket/apollo.yaml
```

Example Python script:

```python
from houston import delete

delete(plan="apollo")
```

Example message:

```json
{
  "command": "delete",
  "plan": "apollo"
}
```

### Start

Starts a new Houston mission by first creating the mission and then starting the first stages. Defaults to starting any
stage that has no upstream dependencies, but can start a specific stage or stages if they are provided in the message.
For convenience, you can also provide stages that should be ignored for the mission.

Example CLI command:

```bash
houston start --plan=apollo
```

Example Python script - starting all stages that don't have upstream dependencies:

```python
from houston import start

start(plan="apollo")
```

starting two specific stages and ignoring a third:

```python
from houston import start

start(plan="apollo", stage=["stage-separation", "refuel"], ignore=["self-destruct"])
```

Example message - starting all stages that don't have upstream dependencies:

```json
{
  "command": "start",
  "plan": "apollo"
}
```

starting two specific stages and ignoring a third:

```json
{
  "command": "start",
  "plan": "apollo",
  "stage": ["stage-separation", "refuel"],
  "ignore": ["self-destruct"]
}
```

### Trigger

Trigger a stage or list of stages. 

Example Python script - triggering a stage in an in-progress mission.

```python
from houston import trigger

trigger(plan="apollo", stage="stage-separation", mission_id="abc123")
```

### Heal (not implemented, will be added in future version)

Example message - healing a specific mission:
```json
{
  "command": "heal",
  "plan": "apollo",
  "mission_id": "abc123"
}
```

healing the latest mission:
```json
{
  "command": "heal",
  "plan": "apollo"
}
```

Similar to trigger but only triggers failed stages. If no mission is specified then the latest mission for the plan is used.  

### Ignore

Ignore the requested stages. If no stages are specified then every stage will be ignored (essentially stopping
the mission. note: Houston cannot stop a stage that has already been started).

Example message - ignoring all stages:
```json
{
  "command": "ignore",
  "plan": "apollo",
  "mission_id": "abc123"
}
```

ignoring specific stages:
```json
{
  "command": "ignore",
  "plan": "apollo",
  "stage": ["self-destruct", "refuel"],
  "mission_id": "abc123"
}
```

Example Python script:

```python
from houston import ignore

ignore(plan="apollo", stages=["self-destruct", "refuel"], mission_id="abc123")
```

### Skip

Mark a stage as completed without running it. Example message:

```json
{
  "command": "skip",
  "plan": "apollo",
  "stage": ["self-destruct", "refuel"],
  "mission_id": "abc123"
}
```

### Fail

Mark a stage as failed. This is useful when a stage needs to be retried but is marked as 'in progress'.

```json
{
  "command": "fail",
  "plan": "apollo",
  "stage": ["self-destruct", "refuel"],
  "mission_id": "abc123"
}
```

### Static Fire

Run requested stage in isolation (ignore dependencies and dependants). This is useful when testing a single stage. 
The service will create a new mission and run the stage in one execution (unlike the _start_ command), therefore the
service used should be the service used by the requested stage. 

Example CLI command:

```bash
houston static-fire --plan=apollo --stage=main-engine-start
```

Example Python script:

```python
from houston import static_fire

static_fire(plan="apollo", stage="main-engine-start")
```

Example message to a Houston service:

```json
{
  "command": "static-fire",
  "plan": "apollo",
  "stage": "main-engine-start"
}
```
