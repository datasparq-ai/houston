# this creates the datasparq/houston-redis image
# see docs/docker for more information

# create a builder enviornment to create houston app
FROM golang:1.18-alpine as builder

WORKDIR /usr/src/houston

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# build houston application
COPY . .
RUN mkdir ./app && go build -v -o ./app ./...

# copy complied houston app to fresh redis image
FROM redis:alpine
COPY --from=builder /usr/src/houston/app/* /usr/local/bin

# start a redis instance
# redis instance will have no password and will not be exposed
COPY --from=builder /usr/src/houston/docker/houston-redis/start.sh /start.sh

RUN chmod +x /start.sh

ENV HOUSTON_PORT=8000

EXPOSE $HOUSTON_PORT 80 443

# run redis in the background then start Houston
# note: the entrypoint is essentially "houston"
ENTRYPOINT ["/start.sh"]
