#!/bin/bash
echo "
Starting the k8s-proxy service and test services.

This script will start 3 simple services.

* service1, service2
    These two services are simple nginx services hosting a page that
    reports which of the two services is being accessed. Neither of
    these services are listening on port 80, that is reserved for the
    proxy service.

    For each of these services, this script will create:
        - a service and deployment for routing traffic and loadbalancing
        pods, listening on port 81.
        - 1 Nginx pod (0.1 CPU, 32Mb ram) that displays the name of it's
        service and deployment.

* k8s-proxy
    This service should serve all traffic on port 80 (working on 443...).
    It will route based on the domain being requested. For example,
    http://service1.somehost should route the deployment managed by
    \`service1\`, and http://service2.somehost should route to the
    deployment managed by \`service2\`.

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

echo
echo "removing service1 deployment and service..."
kubectl delete deploy  service1 > /dev/null
kubectl delete service service1 > /dev/null

echo "removing service2 deployment and service..."
kubectl delete deploy  service2 > /dev/null
kubectl delete service service2 > /dev/null

echo "removing k8s-proxy deployment and service..."
kubectl delete deploy  k8s-proxy > /dev/null
kubectl delete service k8s-proxy > /dev/null
#kubectl delete ingress k8s-proxy

echo
echo "applying service1 deployment and service..."
cat service1.yml | sed s,\$PWD,$(pwd), | kubectl create -f - > /dev/null

echo "applying service2 deployment and service..."
cat service2.yml | sed s,\$PWD,$(pwd), | kubectl create -f - > /dev/null

echo "applying k8s-proxy deployment and service..."
kubectl apply -f k8s-proxy-dev.yml > /dev/null

pod=
printf "\n"
trycount=0
while [ ! -n "$pod" ] && [ "50" -gt "$trycount" ]; do
    sleep 0.5
    pod=$(kubectl get po | grep k8s-proxy | grep -i running | grep '1/1' | awk '{print $1}')
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
echo "Pod:"
echo "$(kubectl get po | egrep '(k8s-proxy)|(NAME)' | grep -v Terminating)"
echo

kubectl logs -f $pod
