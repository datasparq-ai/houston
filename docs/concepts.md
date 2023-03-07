
# Concepts

Running a workflow with Houston involves the following components: 

- Houston API: Orchestrates the services and serves the dashboard UI
- Services: Microservices that execute stages and communicate with the API using the Houston client
- Pub/Sub Messaging System: The tool (of the user's choice) that triggers each microservice to carry out a stage

A Houston workflow is made up of the following concepts:

- Key: Used to authenticate with the API. One key per project
- Plan: The DAG definition. Can be defined in a YAML of JSON file, uploaded to the API, and referred to by name
- Stages: These make up the plan, each runs on a service
- Missions: Individual runs of a plan

### What does the Houston API do?

- Ensures stages run exactly once (deduplicates Pub/Sub messages)
- Figures out which stages can run next by looking at dependencies in the DAG
- Tells the service what to do in order to run the stage (parameters)

### What do services do?

- Tell the API that the stage has started/finished/failed
- Execute a stage in a plan (run the user's code) once per invocation 
- Get the required information from the API to trigger the next stage


### Why use a Pub/Sub Messaging system?

A publisher/subscriber messaging system (such as Google Pub/Sub or Kafka) is used to trigger microservices instead of HTTP for the following reasons:

- Message delivery is guaranteed
- Stages will always be retried if they fail (as the message will be unacknowledged)
- Services don't get overloaded by too many requests. Services can pull messages at their own pace and multiple instances can process messages in parallel
- Multiple missions can use the same services at the same time without needing to wait for the service to become available
