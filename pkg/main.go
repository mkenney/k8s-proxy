package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	logfmt "github.com/mkenney/go-log-fmt"
	"github.com/mkenney/k8s-proxy/pkg/proxy"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

	if "1" == os.Getenv("DEV") || "true" == os.Getenv("DEV") {
		DEV = true
	}

	PORT, err = strconv.Atoi(os.Getenv("PORT"))
	if nil != err || PORT > 65535 {
		log.Warnf("invalid PORT env '%d', defaulting to port 80", PORT)
		PORT = 80
	}

	SECUREPORT, err = strconv.Atoi(os.Getenv("SECUREPORT"))
	if nil != err || SECUREPORT > 65535 {
		log.Warnf("invalid SECUREPORT env '%d', defaulting to port 443", SECUREPORT)
		SECUREPORT = 443
	}

	TIMEOUT, err = strconv.Atoi(os.Getenv("TIMEOUT"))
	if nil != err || TIMEOUT > 900 || TIMEOUT < 0 {
		log.Warnf("invalid TIMEOUT env '%d', defaulting to 10 seconds", TIMEOUT)
		TIMEOUT = 10
	}

	// log level and format
	levelFlag := os.Getenv("LOG_LEVEL")
	if "" == levelFlag {
		levelFlag = "info"
	}
	level, err := log.ParseLevel(levelFlag)
	if nil != err {
		log.Warnf("Could not parse log level flag '%s', setting to 'debug'...", err.Error())
		level, _ = log.ParseLevel("debug")
	}
	log.SetFormatter(&logfmt.TextFormat{})
	log.SetLevel(level)
}

func main() {

	proxy, err := proxy.New(
		DEV,
		PORT,
		SECUREPORT,
		TIMEOUT,
	)
	if nil != err {
		log.Fatal(err)
	}

	errChan := proxy.Start()

	proxy.Wait() // Wait for the k8s services to be ready
	log.Infof("ready to serve traffic")

	// Shutdown when a signal is received.
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c)
		sig := <-c
		log.Warnf("'%s' signal received, shutting down proxy", sig)
		proxy.Stop()
		// Sharing proxy.Start()'s error channel.
		errChan <- fmt.Errorf("'%s' signal received, proxy shut down", sig)
	}()

	go func() {
		list, _ := proxy.K8S.CoreV1().Endpoints("").List(metav1.ListOptions{})
		time.Sleep(5 * time.Second)
		for _, item := range list.Items {
			for _, subset := range item.Subsets {
				tmp, _ := json.MarshalIndent(subset, "", "    ")
				log.Debugf("++++++++++++++++++\n%s - %s\n++++++++++++++++", item.Name, string(tmp))
			}
		}
		//tmp, _ := json.MarshalIndent(list, "", "    ")

		//list, _ := proxy.K8s.Ext.Ingresses("").List(metav1.ListOptions{})
		////item := list.Items[0]
		////fmt.Println(item.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Path)
		//tmp, _ := json.MarshalIndent(list, "", "    ")

		//		log.Debugf(`
		//
		//==========================================
		//%s
		//%+v
		//=======================================
		//
		//`, string(tmp), list)
	}()

	log.Fatal(<-errChan)
}
