FROM golang:1.20

RUN apt-get update && \
    apt-get install -y iputils-ping iperf3
    

WORKDIR /usr/src/app

COPY ./core ../core

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY ./main/go.mod ./main/go.sum ./

RUN go mod download && go mod verify

COPY ./main .
RUN go build -v -o /usr/local/bin/app ./...

#FROM alexdecb/net_utils:latest

CMD ["app"]