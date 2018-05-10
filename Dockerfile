FROM golang:1.10-alpine AS build

ENV DEFAULT_SERVICE=kubernetes \
    K8S_PROXY_PORT=80 \
    K8S_PROXY_SSLPORT=443 \
    K8S_PROXY_TIMEOUT=10

RUN apk update \
    && apk add build-base

WORKDIR /go/src/github.com/mkenney/k8s-proxy/pkg
COPY ./pkg /go/src/github.com/mkenney/k8s-proxy/pkg
RUN GOOS=linux GOARCH=amd64 go build -buildmode=pie -o /go/bin/k8s-proxy

FROM alpine:3.7
LABEL org.label-schema.schema-version = 1.0 \
    org.label-schema.vendor = mkenney@webbedlam.com \
    org.label-schema.vcs-url = https://github.com/mkenney/k8s-proxy \
    org.label-schema.description = "This service provides HTTP ingress proxy functionality for services in a kubernetes cluser." \
    org.label-schema.name = "Kubernetes Ingress Controller" \
    org.label-schema.url = https://github.com/mkenney/k8s-proxy

COPY --from=build /go/bin/k8s-proxy /bin/k8s-proxy
COPY ./assets/k8s-proxy.crt /go/src/github.com/mkenney/k8s-proxy/assets/k8s-proxy.crt
COPY ./assets/k8s-proxy.key /go/src/github.com/mkenney/k8s-proxy/assets/k8s-proxy.key
COPY ./assets/favicon.ico /go/src/github.com/mkenney/k8s-proxy/assets/favicon.ico

#RUN apk update \
#    && apk add build-base

EXPOSE 80
EXPOSE 443
WORKDIR /bin

ENTRYPOINT ["/bin/k8s-proxy"]
