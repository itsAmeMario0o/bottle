# Use this Dockerfile if the version of docker you are using does not support multi-stage builds

FROM centos:7

RUN yum install -y go iptables ipset which redhat-lsb-core openssl net-tools git
RUN touch /.sensor_container

ENV VERSION 1.10.3
ENV FILE go$VERSION.linux-amd64.tar.gz
ENV URL https://storage.googleapis.com/golang/$FILE
ENV SHA256 fa1b0e45d3b647c252f51f5e1204aba049cde4af177ef9f2181f43004f901035

ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

RUN set -eux &&\
  yum -y clean all &&\
  curl -OL $URL &&\
    echo "$SHA256  $FILE" | sha256sum -c - &&\
    tar -C /usr/local -xzf $FILE &&\
    rm $FILE &&\
  mkdir -p "$GOPATH/src" "$GOPATH/bin" && chmod -R 777 "$GOPATH"

WORKDIR $GOPATH/src/ship

COPY ship/ship.go .

RUN go get -d .
RUN go build ship

COPY sensor/sensor.rpm .
COPY sensor/api_credentials.json .

CMD ["./ship"]