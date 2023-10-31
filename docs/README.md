
# Houston Documentation

### Quick links

- Home: [callhouston.io](https://callhouston.io)
- Quickstart: [Python](https://github.com/datasparq-intelligent-products/houston-quickstart-python) (15 minutes), [video](todo) [coming soon]
- Community: [Twitter](https://twitter.com/callhouston_io)
- Contributing
  - [Contributing Guide](./contributing.md)
  - [Kanban Board]() [coming soon]


### Articles

[Concepts](concepts.md) (start here!)

[Services](services.md)
- [Commands](commands.md)
- [Trigger Methods](service_trigger_methods.md)
- [Google Cloud Platform](google_cloud.md)

[API](api.md)
- [Keys](api.md#keys)
- [Config](config.md)
- [Docker](docker.md)
- [Database Schema](database_schema.md)
- [Demo Mode](demo_mode.md)
- [Websocket](websocket.md)
- [Transport Layer Security (TLS) / SSL / HTTPS](./tls.md)
- [Developer Guide](developer_guide.md)
  - [Unit Tests](developer_guide.md#run-unit-tests)
- [API Schema (swagger.json)](https://storage.googleapis.com/houston-static/swagger.json)

Houston Client:
- Python: [PyPi](https://pypi.org/project/houston-client/), [source](https://github.com/datasparq-intelligent-products/houston-python)
- Go [WIP]: [github.com/datasparq-ai/houston/client](https://github.com/datasparq-ai/houston/client)
- Other: [swagger.json](https://storage.googleapis.com/houston-static/swagger.json)

Docker images:
- [docker.io/datasparq/houston](https://hub.docker.com/r/datasparq/houston), [source](../docker/houston/Dockerfile)
- [docker.io/datasparq/houston-redis](https://hub.docker.com/r/datasparq/houston-redis), [source](../docker/houston-redis/Dockerfile)

Terraform modules:
- houston/google: [registry](https://registry.terraform.io/modules/datasparq-ai/houston/google/latest), [source](https://github.com/datasparq-ai/terraform-google-houston)
- houston-key/google: [registry](https://registry.terraform.io/modules/datasparq-ai/houston-key/google/latest), [source](https://github.com/datasparq-ai/terraform-google-houston-key)

Houston UI: [source](https://github.com/datasparq-ai/houston-ui)
