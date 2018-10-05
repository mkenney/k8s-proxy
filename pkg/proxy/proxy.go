package proxy

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"

	errs "github.com/bdlm/errors"
	"github.com/bdlm/log"
	"github.com/mkenney/k8s-proxy/pkg/k8s"
	apiv1 "k8s.io/api/core/v1"
	//"rsc.io/letsencrypt"
)

// New initializes the proxy service and returns a pointer to the service
// instance. If an error is generated while initializing the kubernetes service
// scanner an error will be returned.
func New(
	port int,
	tlsCert string,
) (*Proxy, error) {
	var err error
	proxy := &Proxy{
		listeners: make(map[string]*Listener),
		requestCh: make(chan Request, 15),
		services:  make(map[string]*Service),
	}
	proxy.api, err = k8s.New()
	if nil != err {
		return nil, err
	}
	return proxy, err
}

// Proxy holds configuration data and methods for running the kubernetes proxy
// service.
type Proxy struct {
	api     *k8s.K8S
	port    int
	tlsCert string

	requestCh chan Request
	svcCh     chan apiv1.Service
	stopCh    chan error

	mux       sync.Mutex
	listeners map[string]*Listener
	services  map[string]*Service
}

// AddService adds a service to the passthrough map.
func (proxy *Proxy) AddService(service apiv1.Service) error {
	if service.Name == "k8s-proxy" {
		return fmt.Errorf("'k8s-proxy' cannot be a proxy target, skipping")
	}

	// Service dns hostname.
	host := service.Name
	if _, ok := service.Labels["k8s-proxy-host"]; ok {
		host = service.Labels["k8s-proxy-host"]
	}

	// Make sure service proxy connections exist and are up to date for this
	// service.
	proxy.mux.Lock()
	if _, ok := proxy.services[host]; !ok {
		proxy.services[host] = NewService(service, proxy.api)
	} else {
		proxy.services[host].Refresh()
	}
	proxy.mux.Unlock()

	// Inspect service ports for requirements.
	for _, conn := range proxy.services[host].Conns() {
		mapKey := fmt.Sprintf("%s:%d", conn.Protocol(), conn.Port())

		// Make sure listeners exist for this request channel.
		proxy.mux.Lock()
		_, ok := proxy.listeners[mapKey]
		if !ok {
			proxy.listeners[mapKey] = NewListener(
				conn.Protocol(),
				conn.Port(),
				proxy.requestCh,
			)
		}
		proxy.mux.Unlock()
	}

	return nil
}

// ListenAndServe starts the traffic manager.
func (proxy *Proxy) ListenAndServe() error {

	// update the k8s-proxy service deployment to reflect network+port requirements

	for _, listener := range proxy.listeners {
		listener.Listen()
	}

	for {
		select {
		case <-proxy.stopCh:
			proxy.stopCh <- nil
			return nil
		case request := <-proxy.requestCh:
			// pass the request conn to the correct service
			log.Debug(request)

		// add new services to the
		case svc := <-proxy.svcCh:
			err := proxy.AddService(svc)
			if nil != err {
				return err
			}
		}
	}
}

// RemoveService removes a service from the map.
func (proxy *Proxy) RemoveService(service apiv1.Service) error {
	host := service.Name
	if _, ok := service.Labels["k8s-proxy-hostname"]; ok {
		host = service.Labels["k8s-proxy-hostname"]
	}

	proxy.mux.Lock()
	defer proxy.mux.Unlock()
	if _, ok := proxy.services[host]; !ok {
		return fmt.Errorf("could not remove service '%s', no match found in service map", service.Name)
	}

	log.WithFields(log.Fields{"service": service.Name, "host": host}).
		Info("removing service")
	delete(proxy.services, host)

	return nil
}

// Stop causes the proxy to shutdown. In a kubernetes cluster this will
// cause the container to be restarted.
func (proxy *Proxy) Stop() error {
	e := errs.Err{}

	proxy.mux.Lock()
	defer proxy.mux.Unlock()

	// Stop incomming traffic.
	for _, listener := range proxy.listeners {
		err := listener.Close()
		if nil != err {
			e = e.With(err, "error stopping '%s:%s' listener", listener.Protocol(), listener.Port())
		}
	}

	// Stop service responses.
	for _, service := range proxy.services {
		err := service.Close()
		if nil != err {
			e = e.With(err, "error stopping '%s' service", service.Host())
		}
	}

	// Stop the request manager.
	proxy.stopCh <- nil
	err := <-proxy.stopCh
	if nil != err {
		e = e.With(err, "request listener returned an error on shutdown")
	}

	// Drain orphaned requests and close the channel.
	for 0 < len(proxy.requestCh) {
		<-proxy.requestCh
	}
	close(proxy.requestCh)

	// Drain orphaned service responses and close the channel.
	for 0 < len(proxy.svcCh) {
		<-proxy.svcCh
	}
	close(proxy.svcCh)

	// Stop the k8s service watcher.
	proxy.api.Services.Stop()

	if len(e) > 0 {
		return e
	}
	return nil
}

// UpdateServices processes changes to the set of available services in the
// cluster.
func (proxy *Proxy) UpdateServices(delta k8s.ChangeSet) {
	for _, service := range delta.Added {
		proxy.AddService(service)
	}
	for _, service := range delta.Removed {
		proxy.RemoveService(service)
	}
}

// faviconBytes stores the favicon.ico data.
var faviconBytes []byte

// getFavicon returns the favicon.ico data.
func getFavicon() []byte {
	if nil == faviconBytes || 0 == len(faviconBytes) {
		var err error
		faviconBytes, err = ioutil.ReadFile("/go/src/github.com/mkenney/k8s-proxy/assets/favicon.ico")
		if nil != err {
			log.Error(err)
		}
	}
	return faviconBytes
}

// Pass passes HTTP traffic through to the requested service.
func (proxy *Proxy) Pass(w http.ResponseWriter, r *http.Request) {
	var err error
	scheme := "http"
	if nil != r.TLS {
		scheme = "https"
	}

	//svc, err := proxy.getService(r)
	if nil != err {
		log.WithFields(log.Fields{
			"url": fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL),
			"err": err.Error(),
		}).Warn("request failed, no matching service found")

		if "/favicon.ico" == r.URL.String() {
			w.Header().Set("Content-Type", "image/vnd.microsoft.icon")
			w.WriteHeader(http.StatusOK)
			w.Write(getFavicon())

		} else {
			w.WriteHeader(http.StatusBadGateway)
			HTTPErrs[http.StatusBadGateway].Execute(w, struct {
				Host     string
				Scheme   string
				Services map[string]*Service
			}{
				Host:     r.Host,
				Scheme:   strings.ToUpper(scheme),
				Services: proxy.services,
			})
		}
		return
	}

	//log.WithFields(log.Fields{
	//	"k8s-url": svc.URL(),
	//	"service": svc.K8s().Name,
	//}).Info("serving request")

	// Inject our own ResponseWriter to intercept the result of the
	// proxied request.
	proxyWriter := &ResponseWriter{make([]byte, 0), http.Header{}, 200}
	//svc.Proxy.ServeHTTP(proxyWriter, r)

	// Write headers.
	for k, vals := range proxyWriter.Header() {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	if 502 == proxyWriter.Status() {
		log.WithFields(log.Fields{
			"status": http.StatusText(proxyWriter.Status()),
			"host":   r.Host,
		}).Info("service responded with an error")

		if "/favicon.ico" == r.URL.String() {
			w.Header().Set("Content-Type", "image/vnd.microsoft.icon")
			w.WriteHeader(http.StatusOK)
			w.Write(faviconBytes)

		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			HTTPErrs[http.StatusServiceUnavailable].Execute(w, struct {
				Reason string
				Host   *url.URL
				Msg    string
			}{
				Reason: fmt.Sprintf("%d %s", proxyWriter.Status(), http.StatusText(proxyWriter.Status())),
				//Host:   svc.Proxy.URL,
				Msg: "The deployed pod(s) may be unavailable or unresponsive.",
			})
		}
	} else {
		w.WriteHeader(proxyWriter.Status())
	}
	w.Write(proxyWriter.data)
}
