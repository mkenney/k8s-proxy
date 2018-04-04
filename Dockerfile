FROM golang:1.10

ENV DEFAULT_SERVICE=kubernetes \
    DEV=true \
    PORT=80 \
    SECUREPORT=443 \
    TIMEOUT=10

COPY . /go/src/github.com/mkenney/k8s-proxy
WORKDIR /go/src/github.com/mkenney/k8s-proxy/pkg
RUN go build -o /go/bin/k8s-proxy

EXPOSE 80
EXPOSE 443

ENTRYPOINT ["/go/bin/k8s-proxy"]
