# create a builder enviornment to create houston app
FROM golang:1.18-alpine as builder

WORKDIR /usr/src/houston

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# build houston application
COPY . .
RUN mkdir ./app && go build -v -o ./app ./...

# copy complied houston app to fresh alpine image
FROM alpine:3.17
COPY --from=builder /usr/src/houston/app/* /usr/local/bin

EXPOSE 8000

ENTRYPOINT ["houston"]
