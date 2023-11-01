
# API

The Houston API is a web server that handles orchestration.

It has a REST API for managing keys, plans, and missions. You can interact with it via the CLI, or with one of the Houston 
clients ([python](https://pypi.org/project/houston-client/), [go](https://github.com/datasparq-ai/houston/client)), 
or with a simple HTTP request (see [API Schema (swagger.json)](https://storage.googleapis.com/houston-static/swagger.json)).

## Start a Server

The following starts a server with the default config. This server will have **no password** and so isn't suitable for
a production deployment:

```bash
houston api
```

You can check the health of the server with an API request:

```bash
curl http://localhost:8000/api/v1
```

We recommend reading the [quickstart guide](https://github.com/datasparq-intelligent-products/houston-quickstart-python)
for more information on deploying an API server. 

## Keys

Keys are used to authenticate with the API. We recommend using one key per project/environment.


### Create a Key

With the Houston CLI:

```bash
houston create-key --name "My New Project"
```

This command prints the randomly generated key ID.

If you prefer to choose the key ID yourself:

```bash
houston create-key --name "My New Project" --id foobar1234
```

