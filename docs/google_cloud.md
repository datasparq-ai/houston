
## Houston & Google Cloud Platform

### Google Cloud Functions

[Google Cloud Functions](https://cloud.google.com/functions) are a way to run code with zero server management.  
The `houston.gcp.cloud_function` decorator is a function wrapper that will convert any Python function to a Houston 
service. This service can be executed using a [Pub/Sub](https://cloud.google.com/pubsub) trigger. Example usage:

```python
# main.py

from houston.gcp.cloud_function import service

@service(name="My Service")
def main(param_1, param_2):
    print("hello", param_1, param_2)
```

The above function could be deployed with:

```bash
GCP_PROJECT="my-gcp-project"
HOUSTON_BASE_URL="your Houston API base URL"
HOUSTON_KEY="your project's API key"
gcloud functions deploy my-function-name --runtime python39 --trigger-topic my-function-topic \
    --source . --entry-point main --timeout 540 --set-env-vars "GCP_PROJECT=$GCP_PROJECT,HOUSTON_BASE_URL=$HOUSTON_BASE_URL,HOUSTON_KEY=$HOUSTON_KEY"
```

Note: this also requires a requirements.txt to be present, which must contain `houston-client[gcp]`.

The function can then execute stages, provided the Pub/Sub topic name is provided in the stage params in your plan.
A message like the following would trigger it to run a stage:

```json
{"plan": "my-plan", "stage": "my-stage", "mission_id": "a0234gyil344enbp"}
```

The stage parameters will be provided to the wrapped function as they are defined in the plan. For example, the
stage config for the above example function could look like the following:

```yaml
name: my-stage
service: my-function
params:
  param_1: "foo"
  param_2: 123
```

The service `my-function` should be defined in the plan as having a Pub/Sub trigger:

```yaml
name: my-function
trigger:
  method: google/pubsub
  topic: my-function-topic  # this was set in the `gcloud functions deploy` command
  project: my-project-id    #  optional: defaults to GCP_PROJECT or PROJECT_ID environment variable 
```  

More information on the Pub/Sub trigger can be found in [Service Trigger Methods](./service_trigger_methods.md).

The `houston.gcp.cloud_function.service` wrapper abstracts away the interface between Pub/Sub messages & Houston, 
and the setup of the Houston client, stage parameters, and the triggering of downstream stages. 

A Cloud Function identical to the service shown above without the `service` wrapper could be created with the following 
code:

```python
# main.py

from houston.gcp import Houston

def main(event, context):
    
    # decode and parse Pub/Sub message
    event_data = Houston.extract_stage_information(event['data']) if 'data' in event else event
    
    # initialise client
    client = Houston(plan=event_data['plan'])
    
    # start the stage
    client.start_stage(stage_name=event_data['stage'], mission_id=event_data['mission_id'])
    
    # get the stage params
    params = client.get_params(stage_name=event_data['stage'])
    
    # run the stage
    print("hello", params['param_1'], params['param_2'])

    # end the stage and trigger the downstream stages
    res = client.end_stage(stage_name=event_data['stage'], mission_id=event_data['mission_id'])
    client.trigger_all(res['next'], mission_id=event_data['mission_id'])
```

Note: This version of the service will not be able to run [commands](./commands.md), and will not mark itself as 
failed if there are any errors.

### Providing the API Key

Your Houston API key must be provided to the service for it to be able to make calls to the Houston API. 
It can be provided in the following ways:

1. Secret stored in GCP Secret Manager: The service will automatically look for a secret with name `houston-key` and use the latest value. 
   If you want to name the secret something else, you can set the name with the `HOUSTON_KEY_SECRET_NAME` environment variable.
   The service must have the `secretmanager.secretAccessor` role to read secrets.

2. Environment variable: The API key can be provided via the `HOUSTON_KEY` environment variable.

### Commands

Messages to Houston services can contain Houston [commands](./commands.md). [Commands](./commands.md) are additional high-level methods to allow users or 
Houston integrated services to carry out common tasks with a single command. 
A Houston integrated service will a [command](./commands.md) when triggered with a message containing the 'command' attribute.

A message like the following would save/update a plan:

```json
{"plan": "gs://my-bucket/my-plan.yaml", "command": "save"}
```

The following would start a new mission:

```json
{"plan": "my-plan", "command": "start"}
```

