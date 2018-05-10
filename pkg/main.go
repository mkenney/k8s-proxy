package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"

	logfmt "github.com/mkenney/go-log-fmt"
	"github.com/mkenney/k8s-proxy/pkg/proxy"
	log "github.com/sirupsen/logrus"
)

/*
K8S_PROXY_PORT defines the exposed k8s-proxy port.
*/
var K8S_PROXY_PORT int

/*
K8S_PROXY_SSLPORT defines the exposed k8s-proxy SSL port.
*/
var K8S_PROXY_SSLPORT int

/*
K8S_PROXY_TIMEOUT defines the proxy timeout. Cannot be greater than 15 minutes
(900 seconds).
*/
var K8S_PROXY_TIMEOUT int

func init() {
	var err error

	K8S_PROXY_PORT, err = strconv.Atoi(os.Getenv("K8S_PROXY_PORT"))
	if nil != err || K8S_PROXY_PORT > 65535 {
		log.Warnf("invalid K8S_PROXY_PORT env '%d', defaulting to port 80", K8S_PROXY_PORT)
		K8S_PROXY_PORT = 80
	}

	K8S_PROXY_SSLPORT, err = strconv.Atoi(os.Getenv("K8S_PROXY_SSLPORT"))
	if nil != err || K8S_PROXY_SSLPORT > 65535 {
		log.Warnf("invalid K8S_PROXY_SSLPORT env '%d', defaulting to port 443", K8S_PROXY_SSLPORT)
		K8S_PROXY_SSLPORT = 443
	}

	K8S_PROXY_TIMEOUT, err = strconv.Atoi(os.Getenv("K8S_PROXY_TIMEOUT"))
	if nil != err || K8S_PROXY_TIMEOUT > 900 || K8S_PROXY_TIMEOUT < 0 {
		log.Warnf("invalid K8S_PROXY_TIMEOUT env '%d', defaulting to 10 seconds", K8S_PROXY_TIMEOUT)
		K8S_PROXY_TIMEOUT = 10
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
		K8S_PROXY_PORT,
		K8S_PROXY_SSLPORT,
		K8S_PROXY_TIMEOUT,
	)
	if nil != err {
		log.Fatal(err)
	}

	errChan := proxy.Start()
	proxy.Wait() // Block until the proxy service is ready
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

	for err := range errChan {
		if nil != err {
			proxy.Stop()
			log.Fatal(err)
		}
	}
}
