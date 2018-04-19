#!/bin/bash

IMAGE=mkenney/k8s-proxy:latest
DEPLOYMENT=k8s-proxy

echo "
Starting proxy and test services.

This script will build the binary from the current source and start (or
restart) the proxy service, mounting the binary into the container.

It will also start several simple nginx services to use for testing.
Each one hosts a page that reports the name of the service being
accessed. Each service represents a different test case:

* k8s-proxy
    This service should serve all traffic on port 80 (working on 443...).
    It will route based on the domain being requested. For example,
    http://service1.somehost should route the deployment managed by
    \`service1\`, and http://service2.somehost should route to the
    deployment managed by \`service2\`.

* k8s-proxy-test-1
    No labels defined, traffic routed to the service name. Service
    should be available at http://k8s-proxy-test-1... and should
    result in a page that displays 'k8s-proxy-test-1'.

* k8s-proxy-test-2
    Labels defined, traffic routed to the specified subdomain:
        k8s-proxy-domain: k8s-proxy-test-2-label
        k8s-proxy-protocol: HTTP
    Service should be available at http://k8s-proxy-test-2-label... and
    should result in a page that displays 'k8s-proxy-test-2'.

* k8s-proxy-test-3
    Valid service deployed but no deployment to route traffic to.
    Service is expected to be available at http://k8s-proxy-test-3...
    and should instead result in a 503 error after a 30 second timeout
    period.

* k8s-proxy-test-4
    No service deployed and navigating to http://k8s-proxy-test-4... (or
    any other non-existant service) should immediately result in a 502
    error.

Not for production use. Make sure you're configured for the correct
environment...
"
count=5
while [ "0" -lt "$count" ]; do
    printf "."; ((count-=1)); sleep 1
done
printf "\n"

workdir=$(pwd)

cd $workdir/..
if [ "build" = "$1" ] || [ "--build" = "$1" ]; then
    echo "building image..."
    docker build -t $IMAGE . &> /dev/null
    exit_code=$?
    if [ "0" != "$exit_code" ]; then
        echo "  building image '$IMAGE' failed"
        exit $exit_code
    fi
fi

cd $workdir/../pkg
echo "building k8s-proxy binary"
GOOS=linux go build -o $workdir/bin/k8s-proxy
if [ "0" != "$?" ]; then
    echo "  building binary failed"
    exit 1
fi

echo
echo "removing k8s-proxy-test-1 deployment and service..."
kubectl delete deploy  k8s-proxy-test-1 &> /dev/null
kubectl delete service k8s-proxy-test-1 &> /dev/null

echo "removing k8s-proxy-test-2 deployment and service..."
kubectl delete deploy  k8s-proxy-test-2 &> /dev/null
kubectl delete service k8s-proxy-test-2 &> /dev/null

echo "removing k8s-proxy-test-3 deployment and service..."
kubectl delete service k8s-proxy-test-3 &> /dev/null

echo "removing k8s-proxy deployment and service..."
kubectl delete deploy  k8s-proxy &> /dev/null
kubectl delete service k8s-proxy &> /dev/null
kubectl delete ingress k8s-proxy &> /dev/null
kubectl delete ingress k8s-proxy-ssl &> /dev/null

cd $workdir
echo
echo "applying k8s-proxy-test-1 deployment and service..."
cat k8s-proxy-test-1.yml | sed s,\$PWD,$(pwd), | kubectl create -f - > /dev/null

echo "applying k8s-proxy-test-2 deployment and service..."
cat k8s-proxy-test-2.yml | sed s,\$PWD,$(pwd), | kubectl create -f - > /dev/null

echo "applying k8s-proxy-test-3 deployment and service..."
cat k8s-proxy-test-3.yml | kubectl create -f - > /dev/null

echo "applying k8s-proxy deployment and service..."
cat k8s-proxy-dev.yml | sed s,\$PWD,$(pwd), | kubectl create -f - > /dev/null

pod=
printf "\n"
trycount=0
while [ ! -n "$pod" ] && [ "60" -gt "$trycount" ]; do
    sleep 1
    pod=$(kubectl get po | grep 'k8s-proxy' | grep -i running | grep '1/1' | grep -v 'k8s-proxy-test' | awk '{print $1}')
    printf "."
    ((trycount+=1))
done
printf "\n"

echo
echo "Service:"
echo "$(kubectl get service | egrep '(k8s-proxy)|(NAME)')"
echo
echo "Deployment:"
echo "$(kubectl get deploy | egrep '(k8s-proxy)|(NAME)')"
echo
echo "Pods:"
echo "$(kubectl get po | egrep '(k8s-proxy)|(NAME)' | grep -v Terminating)"
echo

if [ "" = "$pod" ]; then
    echo "Timed out waiting for pod to be ready"
    exit 0
fi

# hide the readiness/liveness probe noise...
echo "Executing \`kubectl logs -f $pod | grep -v 'probe OK'\`..."
echo
kubectl logs -f $pod | grep -v 'probe OK'
