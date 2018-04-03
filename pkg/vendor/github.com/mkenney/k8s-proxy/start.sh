#!/bin/bash
workdir=$(pwd)

cd $workdir/pkg
GOOS=linux go build -o ../bin/k8s-proxy
cd $workdir
kubectl delete deploy k8s-proxy
kubectl delete service k8s-proxy
kubectl apply -f k8s.yml
pod=$(kubectl get po | grep k8s-proxy | grep -i creating | awk '{print $1}')
sleep 3
kubectl logs -f $pod


