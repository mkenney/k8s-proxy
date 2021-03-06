package k8s

import (
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

/*
K8S defines the kubernetes API client
*/
type K8S struct {
	Client   corev1.CoreV1Interface
	Services *Services
}

/*
New is the constructor for the K8S struct.
*/
func New() (*K8S, error) {
	// create the Kubernetes API client
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	k8s := &K8S{Client: client.CoreV1()}
	k8s.Services = &Services{
		client:    k8s.Client.Services(""),
		interrupt: make(chan bool),
		svcMap:    make(chan map[string]apiv1.Service),
	}

	return k8s, nil
}
