FROM golang:1.21 AS build

WORKDIR /usr/src/lpm

COPY go.mod go.sum .

RUN go mod download

COPY . .

RUN go build -v -o /usr/local/bin/lpm-collector ./cmd/collector

FROM ubuntu:latest

WORKDIR /usr/local/bin

COPY --from=build /usr/local/bin/lpm-collector .


RUN apt-get update && \
    apt-get install -y iputils-ping iperf3
    

CMD ["lpm-collector"]