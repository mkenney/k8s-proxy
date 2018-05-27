# k8s-proxy

This project provides a simple HTTP proxy service for easily working with multiple web services in a development environment in [Kubernetes](https://kubernetes.io/).

[Docker image here](https://hub.docker.com/r/mkenney/k8s-proxy/).

<p align="center">
	<a href="https://github.com/mkenney/k8s-proxy/blob/master/LICENSE"><img src="https://img.shields.io/github/license/mkenney/k8s-proxy.svg" alt="MIT License"></a>
	<a href="https://github.com/mkenney/software-guides/blob/master/STABILITY-BADGES.md#alpha"><img src="https://img.shields.io/badge/stability-alpha-f4d03f.svg" alt="Beta"></a>
	<a href="https://travis-ci.org/mkenney/k8s-proxy"><img src="https://travis-ci.org/mkenney/k8s-proxy.svg?branch=master" alt="Build status"></a>
	<a href="https://codecov.io/gh/mkenney/k8s-proxy"><img src="https://img.shields.io/codecov/c/github/mkenney/k8s-proxy/master.svg" alt="Coverage status"></a>
	<a href="https://goreportcard.com/report/github.com/mkenney/k8s-proxy"><img src="https://goreportcard.com/badge/github.com/mkenney/k8s-proxy" alt="Go Report Card"></a>
	<a href="https://github.com/mkenney/k8s-proxy/issues"><img src="https://img.shields.io/github/issues-raw/mkenney/k8s-proxy.svg" alt="Github issues"></a>
	<a href="https://github.com/mkenney/k8s-proxy/pulls"><img src="https://img.shields.io/github/issues-pr/mkenney/k8s-proxy.svg" alt="Github pull requests"></a>
	<a href="https://godoc.org/github.com/mkenney/k8s-proxy"><img src="https://godoc.org/github.com/mkenney/k8s-proxy?status.svg" alt="GoDoc"></a>
</p>


The `k8s-proxy` service will serve all traffic on ports `80` and `443`. SSL traffic on port `443` is encrypted using a self-signed certificate, with all of the associated issues that brings. The exposed ports are configurable in the [`k8s-proxy.yml`](https://github.com/mkenney/k8s-proxy/blob/master/k8s-proxy.yml) file. You must set both the exposed ports in the deployment and service, as well as the `K8S_PROXY_PORT` and `K8S_PROXY_SSLPORT` environment variables in the deployment. Exposing the ports allows them to receive traffic and defining the environment variables tells the proxy service which ports to listen on.

The proxy will route traffic by matching the domain being requested to a service running in the cluster. By default, this is done based on the service name. For example a request for `http://service1.any.host.here` would be routed to a service named 'service1', if it exists.

That is convenient but can be cumbersom in practice. You can also map a subdomain to a particular service by applying labels to the service. All labels are optional:
```yaml
kind: Service
    apiVersion: v1
    metadata:
        name: ui_backend_service
        labels:
            - k8s-proxy-scheme: https     # HTTP scheme to use when addressing this service.
            - k8s-proxy-port:   30021     # Port on the service to send traffic to.
            - k8s-proxy-domain: api.myapp # Subdomain to map this service to.
```
Using labels you can be sure that traffic to `http://api.myapp.any.host.here` and `https://api.myapp.any.host.here` (ssl) will be routed to your service.

## Prerequisites

A properly configured and accessible Kubernetes environment and the `start.sh` script expects the `kubectl` executable to be available in the `$PATH`. Your kube context should be set for the intended environment before executing the script.

## Getting started

Start or restart the proxy service. Listens on ports `80` and `443`.
```
./start.sh
```

Or manually apply the deployment and service.
```
kubectl apply -f k8s-proxy.yml
```

Ports can be configured in the k8s-proxy.yml deployment. You must set both the container ports and the `K8S_PROXY_PORT` and `K8S_PROXY_SSLPORT` environment variables (which inform the proxy executable which ports to bind to).
