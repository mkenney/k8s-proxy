#!/bin/bash
echo "
$0: Start services

This script will start k8s-proxy service.

The `k8s-proxy` service should serve all traffic on port 80 (working on
443...). Ports are not yet configurable. It will route based on the
domain being requested. For example, http://service1.somehost will route
the request to the TCP port exposed by `service1` (port 81), and
http://service2.somehost will route to `service2` on port 82.

Not for production use.
"

kubectl delete deploy k8s-proxy
kubectl delete service k8s-proxy
kubectl apply -f k8s-proxy.yml

pod=$(kubectl get po | grep k8s-proxy | grep -i running | awk '{print $1}')
while [ ! -n "$pod" ]; do
    printf "."
    pod=$(kubectl get po | grep k8s-proxy | grep -i running | awk '{print $1}')
done
printf "\n"

kubectl get po
kubectl logs -f $pod
