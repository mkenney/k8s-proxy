package k8s

//import (
//	apiv1 "k8s.io/api/core/v1"
//	"k8s.io/client-go/kubernetes"
//	//corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
//	//ext "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
//	ext "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
//	"k8s.io/client-go/rest"
//)
//
///*
//K8S defines the kubernetes API client
//*/
//type K8S struct {
//	Client   *kubernetes.Clientset // corev1.CoreV1Interface
//	Ext      ext.ExtensionsV1beta1Interface
//	Services *Services
//}
//
///*
//New is the constructor for the K8S struct.
//*/
//func New() (*K8S, error) {
//	// create the in-cluster config
//	config, err := rest.InClusterConfig()
//	if err != nil {
//		return nil, err
//	}
//
//	// create the client
//	client, err := kubernetes.NewForConfig(config)
//	if err != nil {
//		return nil, err
//	}
//
//	//client.
//	k8s := &K8S{
//		Client: client, //.CoreV1(),
//		Ext:    client.ExtensionsV1beta1(),
//	}
//	k8s.Services = &Services{
//		Client:    k8s.Client.CoreV1().Services(""),
//		Interrupt: make(chan bool),
//		SvcMap:    make(chan map[string]apiv1.Service),
//	}
//
//	return k8s, nil
//}
