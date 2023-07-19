
# Houston Docker Image

The Houston docker image can be used to quickly deploy the Houston API or run the command line tool.

Official Houston docker images are available on Docker Hub: 
- [datasparq/houston-redis:latest](https://hub.docker.com/r/datasparq/houston) (recommended)
- [datasparq/houston:latest](https://hub.docker.com/r/datasparq/houston)

## Quickstart

Use the below commands to pull the container and run the API. 
This assumes you want to set an admin password using environment variables:

```bash
docker run --rm -p 8000:8000 -v ./:/data --env HOUSTON_PASSWORD=change_me datasparq/houston-redis api
```

_Note: The above will fail due to the password not being long enough._

This command includes the following recommended arguments:
- `--rm`: Deletes the container after it finishes running
- `-p 8000:8000`: Binds container port 8000 to your local port 8000, so that the API will be reachable
- `-v ./:/data`: Binds your current working directory to `/data` in the container. This allows redis to save `dump.rdb` files (backups of the whole database). Houston will automatically recover if there's a `dump.rdb` file present
- `--env HOUSTON_PASSWORD=change_me`: Sets the 'HOUSTON_PASSWORD' environment variable in the container, which will be picked up by `houston api` and used to protect all admin routes. Passwords are highly recommended

## Build From Source

From the Houston repository root:

```bash
docker build -f docker/houston/Dockerfile -t houston .
```

Push the image to your container registry:

```bash
docker tag houston <your repository>/houston
docker push <your repository>/houston
```

# Docker Compose

To quickly deploy both Houston and Redis on the same machine, download the example 
[docker-compose.yaml](../docker/houston/docker-compose.yaml) file and run the following commands:

```bash
export HOUSTON_PASSWORD=change_me
docker compose up -d
```

To destroy the deployment:
```bash
docker compose down
```
