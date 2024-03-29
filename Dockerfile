FROM golang:1.21-alpine

WORKDIR /app

RUN apk add git build-base

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -tags musl -buildvcs=false

CMD ./spotify-search-proxy
