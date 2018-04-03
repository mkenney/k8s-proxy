FROM golang:1.10

#RUN apt-get -y update
#RUN apt-get -y install curl iproute2 netbase

RUN go get github.com/sirupsen/logrus \
    && go get k8s.io/api/core/v1 \
    && go get k8s.io/apimachinery/pkg/apis/meta/v1 \
    && go get k8s.io/client-go/... \
    && go get k8s.io/client-go/rest

COPY . /go/src/github.com/mkenney/k8s-proxy
WORKDIR /go/src/github.com/mkenney/k8s-proxy/pkg
RUN go build -o /go/bin/k8s-proxy

EXPOSE 80
EXPOSE 443

ENTRYPOINT ["/go/bin/k8s-proxy"]
