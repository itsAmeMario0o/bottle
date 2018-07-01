FROM golang:latest AS builder

WORKDIR /go/src/bottle
COPY bottle.go .

RUN go get -d .
RUN go build bottle

FROM centos:7

COPY --from=builder /go/src/bottle/bottle .
COPY sensor/sensor.rpm .
COPY sensor/api_credentials.json .

RUN yum install -y iptables ipset which redhat-lsb-core dmidecode openssl net-tools

RUN touch /.sensor_container

CMD ["./bottle"]