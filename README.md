# k8s-proxy

<table><tbody><tr>
    <td>
        <a href="https://github.com/mkenney/k8s-proxy/blob/master/LICENSE"><img src="https://img.shields.io/github/license/mkenney/k8s-proxy.svg" alt="MIT License"></a>
    </td>
    <td rowspan="7">
        This project provides a very simple proxy service for easily working on multiple projects in a development environment in <a href="https://kubernetes.io/" target="_blank">Kubernetes</a>.
        <br><br>
        <a href="https://hub.docker.com/r/mkenney/k8s-proxy/">Docker image here</a>.
        <br><br>
		The <a href="https://github.com/mkenney/k8s-proxy/blob/master/k8s-proxy.yml"><code>k8s-proxy.yml</code></a> file defines a service listening on port <code>80</code> and an associated deployment. The deployment runs a single pod with minimal resource requirements (they could probably be lower) that accesses the `kubernetes` API in the cluster it's running in and proxies all traffic on port 80 to a running service with a defined TCP port who's name is a prefix matching the requested domain.
        <br><br>
		SSL passthrough is still a work in progress.
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

Start or restart the proxy service. Listens on port 80.
```
./start.sh
```

Or manually apply the deployment and service.
```
kubectl apply -f k8s-proxy.yml
```
