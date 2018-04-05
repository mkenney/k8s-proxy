#!/bin/bash

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
