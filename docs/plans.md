
# Plans

Houston plans are definitions of workflows. They are represented in YAML or JSON. Multiple plans can be associated with
one key.

Plan definitions have the following attributes:
- name `string`: Name of the plan
- services `[]Service`: (optional) List of services used by the plan, see [Services](./services.md)
- stages `[]Stage`: List of stages in the plan - see below for details

Here's an example plan definition:

```yaml
name: my-plan

services:
  - name: my-service
    trigger:
      method: pubsub
      topic: topic-for-stage
  - name: my-other-service
    trigger:
      method: http
      url: https://example.com/api/houston

stages:
  - name: blastoff
    service: my-service
    params:
      my_param: foo  
  - name: main-engine-cutoff
    service: my-other-service
    upstream:
      - blastoff
```

Plans should be 'saved' using the Houston client. Saved plans can be referenced by name when starting a mission.

## Stages

Stages have the following attributes:
- name `string`: Name for the stage
- service `string`: Name of the service that this stage runs on
- upstream `[]string`: (optional) List of names of other stages that must be completed before this stage can be started
- downstream `[]string`: (optional) List of names of other stages that can only be started after this stage has finished
- params `object[string]object`: (optional) Mapping of parameter names to parameter values

Parameter values can be strings or nested JSON objects. The Houston client will convert the value to a JSON string
before storing it in Houston's database, and convert it back when it gets used by a stage.


## Missions

Missions are individual runs of a plan. They inherit the same stages as the plan they're created from, 
but also have stateful attributes.

Stages in a mission have the following additional attributes:
- state `enum`: one of the possible stage states, which are `ready`, `started`, `finished`, `failed`, `excluded`, and `skipped`
- start `timestamp`: The time when the stage started
- end `timestamp`: The time when the stage ended

The different stage states have the following meanings:
- ready: Hasn't started
- started: In progress
- finished: Finished successfully 
- failed: Has been started and subsequently failed - it can be started again (retried)
- excluded: Not included in the current mission and won't be run - stages that depend on this stage will not run either
- skipped: The mission will run as if this stage doesn't exist - it won't be run, but it's downstream stages will be

---

Read Next: [Services](./services.md)