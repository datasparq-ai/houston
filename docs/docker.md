
# Houston Docker Image 

The official Houston docker image uses a Go base image and contains Houston and Redis.
It can be used to quickly deploy the Houston API.

## Quickstart

Use the below commands to pull the container and run the API. 
This assumes you want to set an admin password using environment variables:

```bash
docker pull <your repository>/houston
docker cp my_config.yaml <your repository>/houston:/my_config.yaml
docker run -p 8000:8000 --env HOUSTON_PASSWORD=change_me <your repository>/houston api
```

_Note: The above will fail due to the password not being long enough._

## Build From Source

From the Houston repository root:

```bash
docker build -f docker/Dockerfile -t houston .
```