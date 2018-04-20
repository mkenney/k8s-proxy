#!/bin/bash

IMAGE=mkenney/k8s-proxy:latest
DEPLOYMENT=k8s-proxy

printf "
Starting the k8s-proxy service.

The \`k8s-proxy\` service will serve all traffic on a specified port.
Ports are not configurable via this script but can be changed in
\`k8s-proxy.yml\`. You must set both the exposed ports in the deployment
and service as well as the PORT and SECUREPORT environment variables in
the deployment. Exposing the ports allows them to receive traffic and
defining the environment variables tells the proxy serice which ports
to listen on.

The proxy will route traffic by matching the domain being requested to
a service running in the cluster. By default, this is done based on the
service name. For example a request for http://service1.any.host.here
would be routed to a service named 'service1', if it exists.

That's convenient but can be cumbersom in practice however, you may also
apply labels to the service to be used by the proxy:

    kind: Service
    apiVersion: v1
    metadata:
        name: ui_backend_service
        labels:
            -   k8s-proxy-domain: api.myapp
                k8s-proxy-protocol: HTTP

With labels you can be sure that traffic to http://api.myapp.any.host.here
will be routed to your service, but https://api.myapp.any.host.here (ssl)
traffic won't.

Not for production use. Make sure your \`kubectl\` cli is configured for
the intended environment"
count=0
while [ "10" -gt "$count" ]; do
    printf "."; ((count+=1)); sleep 1
done
printf "\n\n"

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
count=0
while [ ! -n "$pod" ] && [ "60" -gt "$count" ]; do
    sleep 1
    pod=$(kubectl get po | grep k8s-proxy | grep -i running | grep '1/1' | awk '{print $1}')
    printf "."
    ((count+=1))
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

# hide the readiness/liveness probe noise...
echo "kubectl logs -f $pod | grep -v 'probe OK'"
echo
kubectl logs -f $pod | grep -v 'probe OK'
