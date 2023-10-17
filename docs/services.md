
# Services

Services carry out the tasks at each stage of a mission. These will typically be your own microservice deployments.
They should be defined in the plan within the `services` block with a name and a trigger method. Each stage should 
reference a service which will run that stage.

The microservices used in a plan can be hosted on any platform and written in any language/runtime. 
For example an AWS Lambda running Python, Google Cloud Function running Go, Azure Function App running C#, and on 
premise HTTP server running PHP could all be used within the same Houston plan.

Services can be triggered via any method (HTTP POST request or webhook is fine). We recommend Pub/Sub messaging systems.

Houston services don't have to be tied to a single plan or use case (and shouldn't be). The service should get all 
the information about the task it's doing from the triggering message and plan parameters. This allows one service to
be used in multiple plans, potentially by multiple teams across an organisation, or by authorised third parties.


### Standard Houston Services

Services that use the [Houston Python Client](https://pypi.org/project/houston-client/), and either a `@service` 
decorator the `execute_service` function, will automatically be able to carry out Houston stages and various other tasks
with no additional configuration. 

The setup for a generic Houston service looks like the following (in Python):

```python
from houston.service import execute_service

def my_stage(param_1, param_2):
    print(hello, param_1, param_2)

def main(event):
    execute_service(
        event=event,
        func=my_stage
    )
```

When using a `@service` decorator, the above code is simplified to:

```python
from houston.gcp.cloud_function import service

@service()
def main(param_1, param_2):
    print(hello, param_1, param_2)
```

The `event` is the message that triggers the service, which can look like the following to trigger a stage:

```json
{
  "plan": "my-plan",
  "mission_id": "m0",
  "stage": "stage-1",
  "ignore_dependencies": false,
  "ignore_dependants": false
}
```

Or the following to run a Houston [command](./commands.md):

```json
{
  "command": "start",
  "plan": "my-plan"
}
```

In the case of triggering a stage, the following happens:
- The incoming message is parsed from JSON
- A Houston client is initialised using the API key and base URL from the environment (`HOUSTON_KEY` and `HOUSTON_BASE_URL` environment variables), and the plan name from the triggering message. 
- The client attempts to start the stage:
  - The API determines whether the stage is allowed to run
  - If not, the function stops here (this prevents duplicate messages from causing unwanted stage executions)
  - If `ignore_dependencies` is set to `true`, Houston will ignore the state of all upstream stages, and they will be marked as excluded
- The client loads the plan from the API and reads the stage parameters 
- The function provided (as `func` to `execute_service`) will be executed using the stage parameters
- If the function fails, the stage is marked as failed
- If the function succeeds, the stage is marked as finished
  - If `ignore_dependants` is set to `true`, Houston will mark all downstream stages as excluded
- The Houston API tells the client which stages should run next (if any)
- The client reads the 'services' section plan to determine how to trigger the next stages
- The client triggers the next stages

All the steps above can be completed without `@service` or `execute_service` by using the relevant Houston client or 
Houston API methods. Stages only need be started and then finished to be considered complete, all other steps are optional. 

For more examples, see: 
- [Google Cloud Functions](./google_cloud.md#google-cloud-functions), [Cloud Function Example](https://github.com/datasparq-intelligent-products/houston-quickstart-python/tree/master/google-cloud)
- [HTTP Service](https://github.com/datasparq-intelligent-products/houston-quickstart-python/tree/master/local)


### Trigger Methods

All trigger methods are described in [Service Trigger Methods](./service_trigger_methods.md), along with the required 
service definition for each trigger method.


