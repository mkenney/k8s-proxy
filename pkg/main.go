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
PORT defines the exposed k8s-proxy port.
*/
var PORT int

/*
SSLPORT defines the exposed k8s-proxy SSL port.
*/
var SSLPORT int

/*
TIMEOUT defines the proxy timeout. Cannot be greater than 15 minutes
(900 seconds).
*/
var TIMEOUT int

func init() {
	var err error

	PORT, err = strconv.Atoi(os.Getenv("PORT"))
	if nil != err || PORT > 65535 {
		log.Warnf("invalid PORT env '%d', defaulting to port 80", PORT)
		PORT = 80
	}

	SSLPORT, err = strconv.Atoi(os.Getenv("SSLPORT"))
	if nil != err || SSLPORT > 65535 {
		log.Warnf("invalid SSLPORT env '%d', defaulting to port 443", SSLPORT)
		SSLPORT = 443
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
		PORT,
		SSLPORT,
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
