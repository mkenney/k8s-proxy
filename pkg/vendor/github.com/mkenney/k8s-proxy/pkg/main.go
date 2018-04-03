package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/mkenney/k8s-proxy/src/proxy"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {

	proxy, err := proxy.New()
	if nil != err {
		panic(err)
	}

	defaultSvc := os.Getenv("DEFAULT_SERVICE")
	if "" != defaultSvc {
		proxy.Default = defaultSvc
	}
	dev := os.Getenv("DEV")
	if "1" == dev || "true" == dev {
		proxy.Dev = true
	}
	port := os.Getenv("PORT")
	if "" != port {
		proxy.Port, _ = strconv.Atoi(port)
	}
	secure := os.Getenv("SECURE")
	if "1" == secure || "true" == secure {
		proxy.Secure = true
	}
	timeout := os.Getenv("TIMEOUT")
	if "" != timeout {
		proxy.Timeout, _ = strconv.Atoi(timeout)
	}

	log.Debugf("starting...")
	err = proxy.Start()
	log.Debugf("started...")
	if nil != err {
		panic(err)
	}
	log.Debugf("didn't panic...")
	tmp, _ := json.MarshalIndent(proxy.K8s.Services.Map(), "", "    ")
	log.Debugf("Service Map: '%s'", string(tmp))
	log.Debugf("sleeping...'")
	time.Sleep(5 * time.Second)
	tmp, _ = json.MarshalIndent(proxy.K8s.Services.Map(), "", "    ")
	log.Debugf("Service Map: '%s'", string(tmp))
	time.Sleep(100 * time.Second)
	return

	// Block until a signal is received.
	c := make(chan os.Signal, 1)
	signal.Notify(c)
	log.Debugf("%s received, shutting down...", <-c)

	return

	////////////////////////////////////////////////////////////////////
	////////////////////////////////////////////////////////////////////
	////////////////////////////////////////////////////////////////////

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	for {
		pods, err := clientset.CoreV1().Services("").List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
		for _, item := range pods.Items {
			fmt.Printf("name: %s, port: %d\n", item.Name, item.Spec.Ports[0].Port)
			tmp, _ := json.MarshalIndent(pods.Items, "", "    ")
			fmt.Println(string(tmp))
			os.Exit(0)
		}

		// Examples for error handling:
		// - Use helper functions like e.g. errors.IsNotFound()
		// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
		_, err = clientset.CoreV1().Pods("default").Get("nginx-56ccc8cc57-htbxl", metav1.GetOptions{})
		if errors.IsNotFound(err) {
			fmt.Printf("Pod not found\n")
		} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
			fmt.Printf("Error getting pod %v\n", statusError.ErrStatus.Message)
		} else if err != nil {
			panic(err.Error())
		} else {
			fmt.Printf("Found pod\n")
		}

		time.Sleep(10 * time.Second)
	}
}
