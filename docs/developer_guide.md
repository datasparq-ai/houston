
# Developer Guide

## Run Unit Tests

Test with development database:
```bash
go test ./...
```

Test with Redis database:
```bash
# remove any existing redis data
rm dump.rdb
# prevent go from using cached test results
go clean -testcache
# create redis db
redis-server &
go test ./...
# stop redis db
kill $!
```


## Generate the API Schema (OpenAPI/Swagger)

A swagger.json file can be generated using [swag](https://github.com/swaggo/swag):

```bash
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g api/router.go
```

## Upgrade Packages

To upgrade Houston to a later version of go, first edit the mod file:
 
```bash
go mod edit -go=1.23
```

Then run all unit tests. Finally, change the go version number in the Dockerfiles in the docker folder. 

To upgrade all packages required by the Houston module:

```bash
go get -u 
go mod tidy
```
