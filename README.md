
# Houston

Open source, API based workflow orchestration tool.

![Houston Flowchart](https://storage.googleapis.com/houston-static/images/houston-flowchart.gif)

- Homepage: [callhouston.io](https://callhouston.io)
- Quickstart guide: [houston-quickstart-python](https://github.com/datasparq-intelligent-products/houston-quickstart-python) 
- Docs: [./docs](./docs/README.md)  

This repo contains the API server, go client, and CLI.


### Example Usage

![Houston CLI](https://storage.googleapis.com/houston-static/images/houston-cli.gif)

Start a local server with the default config: `houston api`

Quickly run an end-to-end example workflow: `houston demo`

Or use the Docker container: `docker run -p 8000:8000 datasparq/houston-redis demo`

See the quickstart for a guide on how to create microservices and complete Houston missions using them:
[quickstart](https://github.com/datasparq-intelligent-products/houston-quickstart-python)


### Install

If you have [go](https://golang.org/doc/install) installed you can install with:

```bash
go install github.com/datasparq-ai/houston
```


### Why Houston?

Houston is a simpler, faster, and cheaper alternative to tools like Airflow.

API based orchestration comes with 5 key advantages: 
1. Code can run on serverless tools: lower cost, less maintenance, infinite scale 
2. The server isn't under heavy load, so can handle hundreds of concurrent missions
3. Pub/Sub message delivery is guaranteed, improving reliability
4. Multiple workflows can share the same task runners, aiding collaboration
5. Task runners can run anywhere in any language, allowing for rapid development with no vendor lock-in


### Contributing 

Please see the [contributing](./docs/contributing.md) guide.

Development of Houston is supported by [Datasparq](https://datasparq.ai).

