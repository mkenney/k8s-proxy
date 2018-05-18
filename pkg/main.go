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
K8SPROXYPORT defines the exposed k8s-proxy port.
*/
var K8SPROXYPORT int

/*
K8SPROXYSSLPORT defines the exposed k8s-proxy SSL port.
*/
var K8SPROXYSSLPORT int

/*
K8SPROXYTIMEOUT defines the proxy timeout. Cannot be greater than 15 minutes
(900 seconds).
*/
var K8SPROXYTIMEOUT int

func init() {
	var err error

	K8SPROXYPORT, err = strconv.Atoi(os.Getenv("K8SPROXYPORT"))
	if nil != err || K8SPROXYPORT > 65535 {
		log.Warnf("invalid K8SPROXYPORT env '%d', defaulting to port 80", K8SPROXYPORT)
		K8SPROXYPORT = 80
	}

	K8SPROXYSSLPORT, err = strconv.Atoi(os.Getenv("K8SPROXYSSLPORT"))
	if nil != err || K8SPROXYSSLPORT > 65535 {
		log.Warnf("invalid K8SPROXYSSLPORT env '%d', defaulting to port 443", K8SPROXYSSLPORT)
		K8SPROXYSSLPORT = 443
	}

	K8SPROXYTIMEOUT, err = strconv.Atoi(os.Getenv("K8SPROXYTIMEOUT"))
	if nil != err || K8SPROXYTIMEOUT > 900 || K8SPROXYTIMEOUT < 0 {
		log.Warnf("invalid K8SPROXYTIMEOUT env '%d', defaulting to 10 seconds", K8SPROXYTIMEOUT)
		K8SPROXYTIMEOUT = 10
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
		K8SPROXYPORT,
		K8SPROXYSSLPORT,
		K8SPROXYTIMEOUT,
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
