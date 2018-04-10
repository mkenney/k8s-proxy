FROM golang:1.10-alpine AS build

ENV DEFAULT_SERVICE=kubernetes \
    DEV=true \
    PORT=80 \
    SECUREPORT=443 \
    TIMEOUT=10

COPY ./pkg /go/src/github.com/mkenney/k8s-proxy/pkg
WORKDIR /go/src/github.com/mkenney/k8s-proxy/pkg
RUN go build -o /go/bin/k8s-proxy

FROM alpine

LABEL org.label-schema.schema-version = 1.0
LABEL org.label-schema.vendor = mkenney@webbedlam.com
LABEL org.label-schema.vcs-url = https://github.com/mkenney/k8s-proxy
LABEL org.label-schema.description = "This service provides HTTP ingress proxy functionality for services in a kubernetes cluser."
LABEL org.label-schema.name = "k8s Proxy"
LABEL org.label-schema.url = https://github.com/mkenney/k8s-proxy

EXPOSE 80
EXPOSE 443
WORKDIR /
COPY --from=build /go/bin/k8s-proxy /go/bin/k8s-proxy
COPY ./server.crt /go/src/github.com/mkenney/k8s-proxy/server.crt
COPY ./server.key /go/src/github.com/mkenney/k8s-proxy/server.key

ENTRYPOINT ["/go/bin/k8s-proxy"]
