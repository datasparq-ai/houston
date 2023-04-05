
# Houston

Houston is an open source, API based workflow orchestration tool.

Documentation [./docs](./docs/README.md)
Homepage: [callhouston.io](https://callhouston.io)

This repo contains the API server, go client, and CLI.

### Install

If you have [go](https://golang.org/doc/install) installed you can build the binary yourself and install with:

```bash
go install github.com/datasparq-ai/houston
```

### Example Usage / Quickstart (1 minute)

Use `houston demo` to quickly run an end-to-end example workflow:

```bash
houston demo
```

Alternatively, start a local Houston server with the default config:

```bash
houston api
```

The server is now running at `localhost:8000`. The Houston client will automatically look for Houston API servers 
running at this location.

(in a separate shell) Create a new Houston key with ID = 'quickstart':

```bash
houston create-key -i quickstart -n "Houston Quickstart"
```

Save this example plan to local file, e.g. 'example_plan.yaml':

```yaml
name: apollo
stages:
  - name: engine-ignition
  - name: engine-thrust-ok
    upstream:
      - engine-ignition
  - name: release-holddown-arms
  - name: umbilical-disconnected
  - name: liftoff
    upstream:
      - engine-thrust-ok
      - release-holddown-arms
      - umbilical-disconnected
```

Start a mission using this plan:

```bash
export HOUSTON_KEY=quickstart
houston start --plan example_plan.yaml
```

Then go to http://localhost:8000. Enter your Houston key 'quickstart'.

You've created a plan and started a mission. You now need a microservice to complete each of the stages in this mission.
See the quickstart for a guide on how to create microservices and complete Houston missions using them:
[quickstart](https://github.com/datasparq-intelligent-products/houston-quickstart-python)

### Contributing 

Please see the [contributing](./docs/contributing.md) guide.

Development of Houston is supported by [Datasparq](https://datasparq.ai).

### Run Unit Tests

Test with development database:
```bash
go test ./...
```

Test with Redis database:
```bash
# remove any existing redis database
rm dump.rdb
# prevent go from using cached test results
go clean -testcache
# create redis db 
redis-server &
go test ./...
# stop redis db
kill $!
```
