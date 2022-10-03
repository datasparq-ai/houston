
# Houston

Houston is an open source, API based workflow orchestration tool.

See our documentation on [callhouston.io](https://callhouston.io/docs).

This repo contains the API server, go client, and CLI.

### Example Usage / Quickstart (1 minute)

Start a local Houston server with the default config:

```bash
houston api
```

(in a separate shell) Create a new Houston key with ID = 'quickstart':

```bash
export HOUSTON_KEY=$(houston create-key -i quickstart)
```

Save this example plan to local file: [example_plan.yaml]()

Start a mission using this plan: 

```bash
houston start --plan ./example_plan.yaml
```

Then go to http://localhost:8000. Enter your Houston key 'quickstart'.

### Install

If you have [go](https://golang.org/doc/install) installed you can build the binary yourself and install with:

```bash
go install github.com/datasparq-ai/houston
```

### Contributing 

Development of Houston is supported by [Datasparq](https://datasparq.ai).

### Run Unit Tests

Test with development database:
```bash
go test ./...
```

Test with Redis database:
```bash
redis-server &
go test ./...
kill $!
```
