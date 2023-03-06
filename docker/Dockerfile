FROM golang:1.18-alpine

WORKDIR /usr/src/houston

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /usr/local/bin ./...

# install gettext utilities
RUN apk add gettext

EXPOSE 8000

ENTRYPOINT ["houston"]
