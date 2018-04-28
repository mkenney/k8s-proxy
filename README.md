# k8s-proxy

<table><tbody><tr>
    <td>
        <a href="https://github.com/mkenney/k8s-proxy/blob/master/LICENSE"><img src="https://img.shields.io/github/license/mkenney/k8s-proxy.svg" alt="MIT License"></a>
    </td>
    <td rowspan="7">
        This project provides a simple proxy service for easily working with multiple web services in a development environment in <a href="https://kubernetes.io/" target="_blank">Kubernetes</a>.
        <br><br>
        <a href="https://hub.docker.com/r/mkenney/k8s-proxy/">Docker image here</a>.
        <br><br>
    </td>
</tr><tr>
    <td>
        <a href="https://github.com/mkenney/software-guides/blob/master/STABILITY-BADGES.md#experimental"><img src="https://img.shields.io/badge/stability-experimental-orange.svg" alt="Experimental"></a>
    </td>
</tr><tr>
    <td width="150">
        <a href="https://travis-ci.org/mkenney/k8s-proxy"><img src="https://travis-ci.org/mkenney/k8s-proxy.svg?branch=master" alt="Build status"></a>
    </td>
</tr><tr>
    <td width="150">
        <a href="https://codecov.io/gh/mkenney/k8s-proxy"><img src="https://img.shields.io/codecov/c/github/mkenney/k8s-proxy/master.svg" alt="Coverage status"></a>
    </td>
</tr><tr>
    <td>
        <a href="https://github.com/mkenney/k8s-proxy/issues"><img src="https://img.shields.io/github/issues-raw/mkenney/k8s-proxy.svg" alt="Github issues"></a>
    </td>
</tr><tr>
    <td>
        <a href="https://goreportcard.com/report/github.com/mkenney/k8s-proxy"><img src="https://goreportcard.com/badge/github.com/mkenney/k8s-proxy" alt="Go Report Card"></a>
    </td>
</tr><tr>
    <td>
        <a href="https://godoc.org/github.com/mkenney/k8s-proxy/pkg"><img src="https://godoc.org/github.com/mkenney/k8s-proxy/pkg?status.svg" alt="GoDoc"></a>
    </td>
</tr></tbody></table>

The `k8s-proxy` service will serve all traffic on ports `80` and `443`. SSL traffic on port `443` is encrypted using a self-signed certificate, with all of the associated issues. The exposed ports are configurable in the [`k8s-proxy.yml`](https://github.com/mkenney/k8s-proxy/blob/master/k8s-proxy.yml) file. You must set both the exposed ports in the deployment and service, as well as the PORT and SECUREPORT environment variables in the deployment. Exposing the ports allows them to receive traffic and defining the environment variables tells the proxy service which ports to listen on.

The proxy will route traffic by matching the domain being requested to a service running in the cluster. By default, this is done based on the service name. For example a request for http://service1.any.host.here would be routed to a service named 'service1', if it exists.

That is convenient but can be cumbersom in practice, so instead you can define the name the proxy uses to match a request to a service by applying labels to the service:
```yaml
kind: Service
    apiVersion: v1
    metadata:
        name: myapp_api_service
        labels:
            k8s-proxy-domain: api.myapp
```
Using labels you can be sure that traffic to http://api.myapp.hostname and https://api.myapp.hostname will be routed to your service.

## Getting started

Start or restart the proxy service. Listens on ports `80` and `443`.
```
./start.sh
```

Or manually apply the deployment and service.
```
kubectl apply -f k8s-proxy.yml
```
