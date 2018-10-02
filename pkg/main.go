package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"

	"github.com/bdlm/log"
	"github.com/mkenney/k8s-proxy/pkg/proxy"
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
K8SPROXYSSLCERT defines the name of an SSL certificate located in the
/ssl directory. Adding a certificate to the proxy requires buiding your
own image (or executing the /test/start-dev.sh script which volmounts
everything).
*/
var K8SPROXYSSLCERT string

/*
K8SPROXYTIMEOUT defines the proxy timeout. Cannot be greater than 15 minutes
(900 seconds).
*/
var K8SPROXYTIMEOUT int

func init() {
	var err error

	K8SPROXYPORT, err = strconv.Atoi(os.Getenv("K8S_PROXY_PORT"))
	if nil != err || K8SPROXYPORT > 65535 {
		log.Warnf("invalid K8S_PROXY_PORT env '%d', defaulting to port 80", K8SPROXYPORT)
		K8SPROXYPORT = 80
	}

	K8SPROXYSSLCERT = os.Getenv("K8S_PROXY_SSL_CERT")
	if "" == K8SPROXYSSLCERT {
		log.Warnf("invalid K8S_PROXY_SSL_CERT env '%d', defaulting to 'k8s-proxy'", K8SPROXYSSLPORT)
		K8SPROXYSSLCERT = "k8s-proxy"
	}

	K8SPROXYSSLPORT, err = strconv.Atoi(os.Getenv("K8S_PROXY_SSL_PORT"))
	if nil != err || K8SPROXYSSLPORT > 65535 {
		log.Warnf("invalid K8S_PROXY_SSL_PORT env '%d', defaulting to port 443", K8SPROXYSSLPORT)
		K8SPROXYSSLPORT = 443
	}

	K8SPROXYTIMEOUT, err = strconv.Atoi(os.Getenv("K8S_PROXY_TIMEOUT"))
	if nil != err || K8SPROXYTIMEOUT > 900 || K8SPROXYTIMEOUT < 0 {
		log.Warnf("invalid K8S_PROXY_TIMEOUT env '%d', defaulting to 10 seconds", K8SPROXYTIMEOUT)
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
	log.SetFormatter(&log.TextFormatter{
		ForceTTY: true,
	})
	log.SetLevel(level)
}

func main() {
	proxy, err := proxy.New(
		K8SPROXYPORT,
		K8SPROXYSSLCERT,
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
