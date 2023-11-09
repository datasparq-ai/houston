
# Service Trigger Methods 

Services can be triggered by any method, provided that all other services used in a mission have the ability 
to do use that method. Some common trigger types are supported by default in the Houston client.  

The below table shows all trigger types and which clients currently support them. Refer to the section on each method
for details on the required fields.

| Method Name      | Required Fields  | Python Client                          | Go Client |
|------------------|------------------|----------------------------------------|-----------|
| google/pubsub    | topic            | yes (requires `houston-client[gcp]`)   | no        |
| azure/event-grid | topic, topic_key | yes (requires `houston-client[azure]`) | no        |
| http             | url              | yes                                    | no        |

*HTTP triggers are not recommended. 

Some triggers support different types of authentication:

| Method Name    | Auth name        | Required auth fields                                                                   | Python Client                | Go Client  |
|----------------|------------------|----------------------------------------------------------------------------------------|------------------------------|------------|
| http           | bearer           | token `string`: the identity token                                                     | yes (added in version 1.4.0) | no         |
| http (planned) | basic (planned)  | username `string`: account username <br> password `string`: account password           | no                           | no         |
| http (planned) | apikey (planned) | key `string`: the API key <br> name `string`: the header name to use, e.g. "X-API-KEY" | no                           | no         |



## Google Cloud Pub/Sub Trigger

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


## Microsoft Azure Event Grid Trigger

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

## HTTP / HTTPS Trigger

HTTP triggers use HTTP or HTTPS POST requests. Optionally, they can also require authentication (see below).

If a service is triggered via HTTP, a response should be returned immediately so that the mission can be carried out
asynchronously. In order to ensure that HTTP trigger requests never block the triggering service, the Houston client
will 'fire-and-forget' the request. This means that the response is always ignored. This can potentially cause issues
where unsuccessful requests aren't noticed.

It is recommended to use a messaging service such as Google Pub/Sub, which has guaranteed delivery, instead of HTTP.

An HTTP triggered service with no authentication could look like the following:

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

### HTTP Bearer Auth

Bearer auth for HTTP triggers is currently only supported in the Python client. 
To use Bearer auth, add `auth: bearer` to the service definition:

```yaml
name: my-plan

services:
  - name: my-service
    trigger: 
      method: http
      url: http://example.com/my-api/do-task
      auth: bearer

stages:
  # stages go here
```

Any service that now wishes to trigger this service (including itself) now needs to provide a bearer token when making 
the request. This is done by providing the token within the `auth` object when initialising the Houston client:

```python
token = get_token()  # generate the token

auth = {
  "my-servce": { "token": token },
}
```
