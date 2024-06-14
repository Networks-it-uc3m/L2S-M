FROM golang:1.21 AS build

WORKDIR /usr/src/lpm

COPY go.mod go.sum .

RUN go mod download

COPY . .

RUN go build -v -o /usr/local/bin/lpm-exporter ./cmd/swm_exporter

FROM ubuntu:latest

WORKDIR /usr/local/bin

COPY --from=build /usr/local/bin/lpm-exporter .

    

CMD ["lpm-exporter"]