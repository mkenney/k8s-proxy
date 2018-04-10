#!/bin/bash
echo "
Starting the k8s-proxy service.

The \`k8s-proxy\` service should serve all traffic on a specified port.
Ports are not yet configurable via this script but can be set in
\`k8s-proxy.yml\`. You must set both the exposed ports in the deployment
and service as well as the PORT and SECUREPORT environment variables in
the deployment.

THe proxy will route based on the domain being requested. For example,
http://service1.somehost will route the request to the TCP port exposed
by \`service1\` and http://service2.somehost will route to \`service2\`
using the internal \`kube-dns\` hostname.

Not for production use.
"

echo "removing k8s-proxy deployment and service..."
kubectl delete deploy  k8s-proxy > /dev/null
kubectl delete service k8s-proxy > /dev/null

echo "applying k8s-proxy deployment and service..."
kubectl apply -f k8s-proxy-dev.yml  > /dev/null

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
