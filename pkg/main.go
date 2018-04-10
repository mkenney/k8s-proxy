package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"

	"github.com/mkenney/k8s-proxy/pkg/proxy"
	log "github.com/sirupsen/logrus"
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
		log.Warnf("invalid DEFAULTSVC env, setting to 'kubernetes'")
		DEFAULTSVC = "kubernetes"
	}

	if "1" == os.Getenv("DEV") || "true" == os.Getenv("DEV") {
		DEV = true
	}

	PORT, err = strconv.Atoi(os.Getenv("PORT"))
	if nil != err || PORT > 65535 {
		log.Warnf("invalid PORT env, setting to 80")
		PORT = 80
	}

	SECUREPORT, err = strconv.Atoi(os.Getenv("SECUREPORT"))
	if nil != err || SECUREPORT > 65535 {
		log.Warnf("invalid SECUREPORT env, setting to 443")
		SECUREPORT = 443
	}

	TIMEOUT, err = strconv.Atoi(os.Getenv("TIMEOUT"))
	if nil != err || TIMEOUT > 900 || TIMEOUT < 0 {
		log.Warnf("invalid TIMEOUT env, setting to 10")
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

	proxy.Wait() // Wait for the k8s services to be ready
	log.Infof("ready to serve traffic")

	// Shutdown when a signal is received.
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c)
		sig := <-c
		log.Infof("'%s' signal received, shutting down proxy", sig)
		proxy.Stop()
		errChan <- fmt.Errorf("'%s' signal received, proxy shut down", sig)
	}()

	log.Fatal(<-errChan)
}
