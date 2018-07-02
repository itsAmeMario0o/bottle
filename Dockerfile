FROM golang:latest AS builder

WORKDIR /go/src/ship
COPY src/ship/ship.go .

RUN go get -d .
RUN go build ship

FROM centos:7

COPY --from=builder /go/src/ship/ship .
COPY sensor/sensor.rpm .
COPY sensor/api_credentials.json .

RUN yum install -y iptables ipset which redhat-lsb-core dmidecode openssl net-tools

RUN touch /.sensor_container

CMD ["./ship"]