
# Service Trigger Methods 

Services can be triggered by any method, provided that all other services used in a mission have the ability 
to do use that method. Some common trigger types are supported by default in the Houston client.  

The below table shows all trigger types and which clients currently support them. Refer to the section on each method
for details on the required fields.

| Method Name      | Required Fields  | Python Client                          | Go Client |
|------------------|------------------|----------------------------------------|-----------|
| google/pubsub    | topic            | yes (requires `houston-client[gcp]`)   |           |
| azure/event-grid | topic, topic_key | yes (requires `houston-client[azure]`) |           |
| http*            | url              | yes                                    |           |

*HTTP triggers are not recommended. 

### Google Cloud Pub/Sub Trigger

The two examples below are equivalent; method can be either `pubsub` or `google/pubsub`, topic can contain project or
the project can be provided separately. 

```yaml
- name: my-cloud-function
  trigger:
    method: google/pubsub
    topic: projects/my-gcp-project/topics/my-pubsub-topic
    
- name: my-cloud-function
  trigger:
    method: pubsub
    topic: my-pubsub-topic   # topic ID
    project: my-gcp-project  # (optional) defaults to the value of 'GCP_PROJECT' environment variable
```


### Microsoft Azure Event Grid Trigger

[Event Grid](https://docs.microsoft.com/en-us/azure/event-grid/overview)

```yaml
name: my-plan

services:
  - name: my-service
    trigger: 
      method: azure/event-grid
      topic: topic1.westus2-1.eventgrid.azure.net
      topic_key: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

stages:
  # stages go here
```

##### HTTP Trigger

HTTP triggers use HTTP POST requests.

```yaml
name: my-plan

services:
  - name: my-service
    trigger: 
      method: http
      url: http://example.com/my-api/do-task

stages:
  # stages go here
```

The houston client adds the following request body when triggering via an HTTP POST request:
```json
{
  "plan": "my-plan",
  "mission_id": "m0",
  "stage": "stage-1",
  "ignore_dependencies": false,
  "ignore_dependants": false
}
```

If a service is triggered via HTTP, a response should be returned immediately so that the mission can be carried out 
asynchronously. In order to ensure that HTTP trigger requests never block the triggering service, the Houston client 
will 'fire-and-forget' the request. This means that the response is always ignored. This can potentially cause issues 
where unsuccessful requests aren't noticed. 

It is recommended to use a messaging service such as Google Pub/Sub, which has guaranteed delivery, instead of HTTP.
