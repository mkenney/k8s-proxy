package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/mkenney/k8s-proxy/pkg/proxy"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

/*
DEFAULTSVC defines the default k8s service.
*/
var DEFAULTSVC string

/*
DEV assumes a dev environment if true.
*/
var DEV bool

/*
PORT defines the exposed k8s-proxy port.
*/
var PORT int

/*
SECUREPORT defines the exposed k8s-proxy SSL port.
*/
var SECUREPORT int

/*
TIMEOUT defines the proxy timeout. Cannot be greater than 15 minutes
(900 seconds).
*/
var TIMEOUT int

func init() {
	var err error

	DEFAULTSVC = os.Getenv("DEFAULTSVC")
	if "" == DEFAULTSVC {
		DEFAULTSVC = "kubernetes"
	}

	if "1" == os.Getenv("DEV") || "true" == os.Getenv("DEV") {
		DEV = true
	}

	PORT, err = strconv.Atoi(os.Getenv("PORT"))
	if nil != err || PORT > 65535 {
		PORT = 80
	}

	SECUREPORT, err = strconv.Atoi(os.Getenv("SECUREPORT"))
	if nil != err || SECUREPORT > 65535 {
		SECUREPORT = 443
	}

	TIMEOUT, err = strconv.Atoi(os.Getenv("TIMEOUT"))
	if nil != err || TIMEOUT > 900 {
		TIMEOUT = 10
	}
}

func main() {

	proxy, err := proxy.New(
		DEFAULTSVC,
		DEV,
		PORT,
		SECUREPORT,
		TIMEOUT,
	)
	if nil != err {
		log.Fatal(err)
	}

	errChan := proxy.Start()

	//tmp, _ := json.MarshalIndent(proxy.Map(), "", "    ")
	//log.Debugf("Service Map: '%s'", string(tmp))

	// Shutdown when signal is received.
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c)
		sig := <-c
		log.Infof("'%s' signal received, shutting down proxy", sig)
		proxy.Stop()
		errChan <- fmt.Errorf("'%s' signal received, proxy shut down", sig)
	}()

	log.Fatal(<-errChan)

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
