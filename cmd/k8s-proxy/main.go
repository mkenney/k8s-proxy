package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/bdlm/log"
	"github.com/mkenney/k8s-proxy/pkg/proxy"
)

// K8SPROXYPORT defines the exposed k8s-proxy port.
var K8SPROXYPORT int

// K8SPROXYSSLPORT defines the exposed k8s-proxy SSL port.
var K8SPROXYSSLPORT int

// K8SPROXYSSLCERT defines the name of an SSL certificate located in the
// /ssl directory. Adding a certificate to the proxy requires buiding your
// own image (or executing the /test/start-dev.sh script which volmounts
// everything).
var K8SPROXYSSLCERT string

// K8SPROXYTIMEOUT defines the proxy timeout. Cannot be greater than 15 minutes
// (900 seconds).
var K8SPROXYTIMEOUT int

func init() {
	// log level and format
	levelFlag := os.Getenv("LOG_LEVEL")
	if "" == levelFlag {
		levelFlag = "info"
	}
	level, err := log.ParseLevel(levelFlag)
	if nil != err {
		log.WithField("err", err).Warnf("%-v", err)
		level, _ = log.ParseLevel("debug")
	}
	log.SetFormatter(&log.TextFormatter{
		ForceTTY: true,
	})
	log.SetLevel(level)
}

func main() {
	proxy, err := proxy.New(context.Background())
	if nil != err {
		log.Fatalf("%-v", err)
	}

	log.Infof("starting services...")
	proxy.ListenAndServe()

	// Shutdown when a signal is received.
	c := make(chan os.Signal, 1)
	signal.Notify(c)
	go func() {
		sig := <-c
		log.Infof("'%s' signal received, shutting down proxy", sig)
		proxy.Stop()
	}()

	// debug
	select {
	case <-time.After(10 * time.Second):
		log.Infof("stopping services...")
		proxy.Stop()
	}
}
