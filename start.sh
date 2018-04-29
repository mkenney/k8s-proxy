#!/bin/bash

IMAGE=mkenney/k8s-proxy:latest
DEPLOYMENT=k8s-proxy

k8s_context=$(kubectl config view -o=jsonpath='{.current-context}')
k8s_namespace=$(kubectl config view -o=jsonpath="{.contexts[?(@.name==\"$kcontext\")].context.namespace}")
if [ "" = "$k8s_namespace" ]; then
    k8s_namespace="default"
fi

HIGHLIGHT=$'\033[38;5;172m'
NORMAL=$'\033[0m'
printf "
This script will start the kubernetes proxy service using the \`kubectl apply\`
command. Make sure you are configured for the correct environment.

Current context:   ${HIGHLIGHT}${k8s_context}${NORMAL}
Current namespace: ${HIGHLIGHT}${k8s_namespace}${NORMAL}

"
read -p "Do you want to continue? [y/N]: " EXECUTE
if [ "y" != "$EXECUTE" ] && [ "Y" != "$EXECUTE" ]; then
    exit 0
fi

printf "
Starting the k8s-proxy service.

The \`k8s-proxy\` service will serve all traffic on ports 80 and 443.
SSL traffic is terminated using a self-signed certificate with the
associated issues. Ports are not configurable via this script but can be
changed in \`k8s-proxy.yml\`. You must set both the exposed ports in the
deployment and service as well as the PORT and SECUREPORT environment
variables in the deployment. Exposing the ports allows them to receive
traffic and defining the environment variables tells the proxy serice
which ports to listen on.

Not for production use.

"

if [ "build" = "$1" ] || [ "--build" = "$1" ]; then
    echo "building image..."
    docker build -t $IMAGE . &> /dev/null
    exit_code=$?
    if [ "0" != "$exit_code" ]; then
        echo "  building image '$IMAGE' failed"
        exit $exit_code
    fi
fi

echo "removing k8s-proxy deployment and service..."
kubectl delete deploy k8s-proxy &> /dev/null
kubectl delete service k8s-proxy &> /dev/null
kubectl delete ingress k8s-proxy &> /dev/null

echo "applying k8s-proxy deployment and service..."
kubectl apply -f k8s-proxy.yml > /dev/null

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
echo "kubectl logs -f $pod | grep -v 'probe OK'"
echo
kubectl logs -f $pod | grep -v 'probe OK'
