FROM golang:latest AS builder

WORKDIR /go/src/ship
COPY ship/ship.go .

RUN go get -d .
RUN go build ship

FROM centos:7

RUN yum install -y iptables ipset which redhat-lsb-core dmidecode openssl net-tools

COPY --from=builder /go/src/ship/ship .
COPY sensor/sensor.rpm .
COPY sensor/api_credentials.json .

RUN touch /.sensor_container

CMD ["./ship"]