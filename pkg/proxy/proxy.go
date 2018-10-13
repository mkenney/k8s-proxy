package proxy

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	errs "github.com/bdlm/errors"
	"github.com/bdlm/log"
	"github.com/mkenney/k8s-proxy/internal/codes"
	"github.com/mkenney/k8s-proxy/pkg/k8s"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"rsc.io/letsencrypt"
)

// New initializes the proxy service and returns a pointer to the service
// instance. If an error is generated while initializing the kubernetes service
// scanner an error will be returned.
func New(ctx context.Context) (*Proxy, error) {
	var err error
	ctx, cancel := context.WithCancel(ctx)
	proxy := &Proxy{
		cancelContext: cancel,
		ctx:           ctx,
		listeners:     make(map[string]*Listener),
		requestCh:     make(chan Request, 15),
		services:      make(map[string]*Service),
		svcCh:         make(chan apiv1.Service, 15),
	}
	proxy.api, err = k8s.New()
	if nil != err {
		return nil, errs.Wrap(err, 0, "could not create k8s api connection")
	}
	proxy.svc, err = proxy.api.Client.Services("default").Get("k8s-proxy", metav1.GetOptions{})
	if nil != err {
		return nil, errs.Wrap(err, 0, "could not load the k8s-proxy service")
	}
	return proxy, err
}

// Proxy holds configuration data and methods for running the kubernetes proxy
// service.
type Proxy struct {
	api           *k8s.K8S
	cancelContext context.CancelFunc
	ctx           context.Context
	done          bool
	port          int
	svc           *apiv1.Service
	tlsCert       string

	requestCh chan Request
	svcCh     chan apiv1.Service

	mux       sync.Mutex
	listeners map[string]*Listener
	services  map[string]*Service

	wg sync.WaitGroup
}

// HasListener returns a bool noting whether the specified listener exists
// in the network map.
func (proxy *Proxy) HasListener(key string) bool {
	proxy.mux.Lock()
	_, ok := proxy.listeners[key]
	proxy.mux.Unlock()
	return ok

}

// AddListener adds a protocol listener to the network map.
//
// An error is returned if a matching Listener already exists or a new
// Listener cannot be created.
func (proxy *Proxy) AddListener(
	protocol string,
	port int32,
) error {
	var mapKey string

	for _, proxyPort := range proxy.svc.Spec.Ports {
		// Determine network protocol.
		protocol := "tcp"
		if "" != proxyPort.Protocol {
			protocol = strings.ToLower(string(proxyPort.Protocol))
		}

		// Determine port for receiving traffic.
		port := proxyPort.Port
		if 0 > proxyPort.TargetPort.IntVal {
			port = proxyPort.TargetPort.IntVal
		}
		if 0 > proxyPort.NodePort {
			port = proxyPort.NodePort
		}

		// Check to see if listener exists.
		mapKey = fmt.Sprintf("%s:%d", protocol, port)
		proxy.mux.Lock()
		if _, ok := proxy.listeners[mapKey]; ok {
			proxy.mux.Unlock()
			err := errs.New(codes.NetworkListenerExists, "network listner already exists, skipping")
			log.Debugf("error code: %d", err.Code())
			return err
		}
		proxy.mux.Unlock()
	}

	// Listener does not exist, create.
	listener, err := NewListener(
		proxy.ctx,
		protocol,
		port,
		proxy.requestCh,
		proxy.api.Client.Services("default"),
	)
	if nil != err {
		return errs.Wrap(err, 1, "could not create new listener")
	}

	proxy.mux.Lock()
	proxy.listeners[mapKey] = listener
	proxy.mux.Unlock()

	return nil
}

// AddService adds a service to the passthrough map.
//
// An error is returned if the specified service is the k8s-proxy service or
// a listener does not exist and cannot be provided.
func (proxy *Proxy) AddService(service apiv1.Service) error {
	if "k8s-proxy" == service.Name || "kubernetes" == service.Name {
		return fmt.Errorf("'%s' cannot be a targeted service, skipping", service.Name)
	}
	if "kube-system" == service.Namespace || "docker" == service.Namespace {
		return fmt.Errorf("'%s' is a reserved namespace, skipping", service.Namespace)
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
		proxy.services[host] = NewService(proxy.ctx, service, proxy.api)
	} else {
		proxy.services[host].Refresh()
	}
	proxy.mux.Unlock()

	// Inspect service ports for requirements.
	for _, conn := range proxy.services[host].Conns() {
		mapKey := fmt.Sprintf("%s:%d", conn.Protocol(), conn.Port())

		// Make sure listeners exist for this request channel.
		_, ok := proxy.listeners[mapKey]
		if !ok {
			err := proxy.AddListener(conn.Protocol(), conn.Port())
			if nil != err {
				if e, ok := err.(*errs.Err); !ok {
					err = errs.Wrap(err, 0, "could not add listener")
				} else if e.Code() != codes.NetworkListenerExists {
					err = errs.Wrap(err, e.Code(), "could not add listener")
				}
				return err
			}
			proxy.mux.Lock()
			err = proxy.listeners[mapKey].Listen()
			proxy.mux.Unlock()
			if nil != err {
				if e, ok := err.(*errs.Err); !ok {
					err = errs.Wrap(err, 0, "could not start listener")
				} else if e.Code() != codes.NetworkListenerListening {
					err = errs.Wrap(err, e.Code(), "could not start listener")
				}
				return err
			}
		}
	}

	return nil
}

// Done returns the proxy done flag.
func (proxy *Proxy) Done() bool {
	return proxy.done
}

// ListenAndServe starts the proxy traffic manager.
//
// First, a goroutine watching for kubernetes service deployment changes is
// started which adds and removes entries from the service map stay in sync
// with the k8s environment.
//
// Second, the primary request listen loop is started which watches for
// incomming requests for all services and routes them to the correct
// service.
func (proxy *Proxy) ListenAndServe() {
	log.Info("starting the proxy service...")

	// update the k8s-proxy service deployment to reflect network+port requirements

	// Start the change watcher and the updater. This will block until
	// service data is available from the Kubernetes API.
	changes := proxy.api.Services.Watch(5 * time.Second)
	go func() {
		log.Info("starting the change watcher...")
		proxy.wg.Add(1)
		ctx, cancel := context.WithCancel(proxy.ctx)
		for {
			select {
			case <-ctx.Done():
				log.Info("stopping the change watcher...")
				cancel()
				proxy.wg.Done()
				return
			case delta := <-changes:
				proxy.UpdateServices(delta)
			}
		}
	}()

	go func() {
		log.Info("starting the request watcher...")
		proxy.wg.Add(1)
		ctx, cancel := context.WithCancel(proxy.ctx)
		for {
			select {
			case <-ctx.Done():
				log.Info("stopping the request watcher...")
				cancel()
				proxy.wg.Done()
				return
			case request := <-proxy.requestCh:
				err := proxy.routeRequest(request)
				if nil != err {
					log.WithField("error", err).Errorf("%-v", err)
				}
			}
		}
	}()
}

func (proxy *Proxy) routeRequest(request Request) error {
	return nil
}

// RemoveService removes a service from the map.
func (proxy *Proxy) RemoveService(service apiv1.Service) error {
	host := service.Name
	if _, ok := service.Labels["k8s-proxy-host"]; ok {
		host = service.Labels["k8s-proxy-host"]
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

// Stop causes the proxy to gracefully shutdown. In a kubernetes cluster
// this will cause the container to be restarted.
func (proxy *Proxy) Stop() {

	// Stop the k8s service watcher.
	proxy.api.Services.Stop()

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

	// Cancel the context thread for this instance.
	proxy.cancelContext()

	proxy.done = true
	log.Info("stopping the proxy service...")
	proxy.wg.Wait()
	log.Info("done.")
}

// UpdateServices processes changes to the set of available services in the
// cluster.
func (proxy *Proxy) UpdateServices(delta k8s.ChangeSet) {
	for _, service := range delta.Added {
		err := proxy.AddService(service)
		if nil != err {
			log.Warn(err)
		}
	}
	for _, service := range delta.Removed {
		err := proxy.RemoveService(service)
		if nil != err {
			log.Warn(err)
		}
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
			log.Errorf("%-v", err)
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
