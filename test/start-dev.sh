#!/bin/bash
echo "
$0: Start test services

This script will start 3 simple services.

* service1, service2
    These two services are simple nginx services hosting a page that
    reports which of the two services is being accessed. Neither of
    these services may are listening on port 80, that is reserved for
    the proxy service.

    - 1 service for routing traffic to pods (ea)
    - 3 pods (ea)
    - 0.1 CPU (ea)
    - 32Mb ram (ea)
    - Ports 81 and 82, respectively

* k8s-proxy
    This service should serve all traffic on port 80 (working on 443...).
    It will route based on the domain being requested. For example,
    http://service1.somehost should route the request to the TCP port
    exposed by \`service1\` (port 81), and http://service2.somehost
    should route to \`service2\` on port 82.

Not for production use.
"

workdir=$(pwd)

cd $workdir/../pkg
echo "building k8s-proxy binary"
GOOS=linux go build -o $workdir/bin/k8s-proxy
if [ "0" != "$?" ]; then
    exit 1
fi

cd $workdir

kubectl delete deploy  service1
kubectl delete service service1

kubectl delete deploy  service2
kubectl delete service service2

kubectl delete deploy  k8s-proxy
kubectl delete service k8s-proxy

cat service1.yml | sed s,\$PWD,$(pwd), | kubectl create -f -
cat service2.yml | sed s,\$PWD,$(pwd), | kubectl create -f -

kubectl apply -f k8s-proxy-dev.yml

pod=
printf "\n"
while [ ! -n "$pod" ]; do
    printf "."
    pod=$(kubectl get po | grep k8s-proxy | grep -i running | awk '{print $1}')
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

kubectl logs -f $pod
