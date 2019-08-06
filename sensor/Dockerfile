FROM golang:latest AS builder

WORKDIR /go/src/ship
COPY ship/ship.go .
COPY ship/ui.html .

RUN go get -d .
RUN go build ship

FROM centos:7

RUN yum install -y iptables ipset which redhat-lsb-core openssl net-tools

RUN touch /.sensor_container

COPY sensor/sensor.rpm .
COPY sensor/api_credentials.json .
COPY sensor/launch_ship.sh .

COPY --from=builder /go/src/ship/ship .
COPY --from=builder /go/src/ship/ui.html .

CMD ["./ship"]