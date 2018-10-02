package proxy

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bdlm/log"
	"github.com/mkenney/k8s-proxy/pkg/k8s"
	"github.com/pkg/errors"
	apiv1 "k8s.io/api/core/v1"
	//"rsc.io/letsencrypt"
)

/*
New initializes the proxy service and returns a pointer to the service
instance. If an error is generated while initializing the kubernetes
service scanner an error will be returned.
*/
func New(
	port int,
	sslCert string,
	sslPort int,
	timeout int,
) (*Proxy, error) {
	var err error
	proxy := &Proxy{
		Port:    port,
		SSLCert: sslCert,
		SSLPort: sslPort,
		Timeout: timeout,

		readyCh:    make(chan struct{}, 2),
		serviceMap: make(map[string]*Service),
	}
	proxy.K8s, err = k8s.New()
	return proxy, err
}

/*
Proxy holds configuration data and methods for running the kubernetes
proxy service.
*/
type Proxy struct {
	K8s     *k8s.K8S
	Port    int
	SSLCert string
	SSLPort int
	Timeout int

	ready      bool
	readyCh    chan struct{}
	svcMapMux  sync.Mutex
	serviceMap map[string]*Service
}

/*
Service defines a k8s service proxy.
*/
type Service struct {
	Name   string
	Port   int32
	Scheme string
	Proxy  *ReverseProxy
}

/*
AddService adds a service to the passthrough map.
*/
func (proxy *Proxy) AddService(service apiv1.Service) error {
	if service.Name == "k8s-proxy" {
		return fmt.Errorf("'k8s-proxy' cannot be a proxy target, skipping")
	}

	// Service subdomain
	domain := service.Name
	if _, ok := service.Labels["k8s-proxy-domain"]; ok {
		domain = service.Labels["k8s-proxy-domain"]
	}

	for _, servicePort := range service.Spec.Ports {
		// Service port
		port := servicePort.Port
		if 0 > servicePort.TargetPort.IntVal {
			port = service.Spec.Ports[0].TargetPort.IntVal
		}
		if 0 > servicePort.NodePort {
			port = service.Spec.Ports[0].NodePort
		}
		if p, ok := service.Labels["k8s-proxy-port"]; ok {
			ptmp, err := strconv.Atoi(p)
			if nil != err {
				log.Warn(errors.Wrap(err, fmt.Sprintf("invalid 'k8s-proxy-port' service label '%s'", p)))
			}
			if nil == err {
				port = int32(ptmp)
			}
		}

		if port >= 80 && "TCP" == servicePort.Protocol {
			// HTTP Scheme
			scheme := "http"
			if 443 == servicePort.Port {
				scheme = "https"
			}
			if _, ok := service.Labels["k8s-proxy-scheme"]; ok {
				scheme = service.Labels["k8s-proxy-scheme"]
			}

			rp, err := NewReverseProxy(scheme, service, port)
			if nil != err {
				return err
			}

			log.WithFields(log.Fields{
				"port":    port,
				"service": service.Name,
				"url":     fmt.Sprintf("%s://%s", scheme, domain),
			}).Info("registering service")

			svc := &Service{
				Name:   service.Name,
				Port:   port,
				Scheme: scheme,
				Proxy:  rp,
			}
			proxy.svcMapMux.Lock()
			proxy.serviceMap[domain] = svc
			proxy.svcMapMux.Unlock()

			break
		}
	}
	return nil
}

/*
Map returns a map of the current kubernetes services.
*/
func (proxy *Proxy) Map() map[string]apiv1.Service {
	return proxy.K8s.Services.Map()
}

/*
getService will attempt ot match a request to the correct service.
*/
func (proxy *Proxy) getService(r *http.Request) (*Service, error) {
	match := ""
	for k := range proxy.serviceMap {
		if r.Host == k {
			match = k
			break
		} else if strings.HasPrefix(r.Host, k+".") && len(match) < len(k) {
			match = k
		}
	}
	if "" != match {
		return proxy.serviceMap[match], nil
	}
	return nil, fmt.Errorf("no service found to fulfill request '%s'", r.Host)
}

/*
faviconBytes stores the favicon.ico data.
*/
var faviconBytes []byte

/*
getFaviconBytes returns the favicon.ico data.
*/
func getFaviconBytes() []byte {
	if nil == faviconBytes || 0 == len(faviconBytes) {
		var err error
		faviconBytes, err = ioutil.ReadFile("/go/src/github.com/mkenney/k8s-proxy/assets/favicon.ico")
		if nil != err {
			log.Error(err)
		}
	}
	return faviconBytes
}

/*
Pass passes HTTP traffic through to the requested service.
*/
func (proxy *Proxy) Pass(w http.ResponseWriter, r *http.Request) {
	scheme := "http"
	if nil != r.TLS {
		scheme = "https"
	}

	svc, err := proxy.getService(r)
	if nil != err {
		log.WithFields(log.Fields{
			"url": fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL),
			"err": err.Error(),
		}).Warn("request failed, no matching service found")

		if "/favicon.ico" == r.URL.String() {
			w.Header().Set("Content-Type", "image/vnd.microsoft.icon")
			w.WriteHeader(http.StatusOK)
			w.Write(getFaviconBytes())

		} else {
			w.WriteHeader(http.StatusBadGateway)
			HTTPErrs[http.StatusBadGateway].Execute(w, struct {
				Host     string
				Scheme   string
				Services map[string]*Service
			}{
				Host:     r.Host,
				Scheme:   strings.ToUpper(scheme),
				Services: proxy.serviceMap,
			})
		}
		return
	}

	log.WithFields(log.Fields{
		"endpoint": svc.Proxy.URL,
		"host":     r.Host,
		"url":      r.URL,
	}).Info("serving request")

	// Inject our own ResponseWriter to intercept the result of the
	// proxied request.
	proxyWriter := &ResponseWriter{make([]byte, 0), http.Header{}, 200}
	svc.Proxy.ServeHTTP(proxyWriter, r)

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
				Host:   svc.Proxy.URL,
				Msg:    "The deployed pod(s) may be unavailable or unresponsive.",
			})
		}
	} else {
		w.WriteHeader(proxyWriter.Status())
	}
	w.Write(proxyWriter.data)
}

/*
RemoveService removes a service from the map.
*/
func (proxy *Proxy) RemoveService(service apiv1.Service) error {
	domain := service.Name
	if _, ok := service.Labels["k8s-proxy-domain"]; ok {
		domain = service.Labels["k8s-proxy-domain"]
	}

	if _, ok := proxy.serviceMap[domain]; !ok {
		return fmt.Errorf("could not remove service '%s', no match found in service map", service.Name)
	}

	log.WithFields(log.Fields{
		"service": service.Name,
		"domain":  domain,
	}).Info("removing service")
	proxy.svcMapMux.Lock()
	delete(proxy.serviceMap, domain)
	proxy.svcMapMux.Unlock()

	return nil
}

/*
Start starts the proxy services.
*/
func (proxy *Proxy) Start() chan error {
	errs := make(chan error)

	// Set the global timeout.
	http.DefaultClient.Timeout = time.Duration(proxy.Timeout) * time.Second

	// Start the change watcher and the updater. This will block until
	// service data is available from the Kubernetes API.
	changes := proxy.K8s.Services.Watch(5 * time.Second)
	go func() {
		for delta := range changes {
			proxy.UpdateServices(delta)
		}
	}()

	// Kubernetes liveness probe handler.
	http.HandleFunc("/k8s-alive", func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(log.Fields{
			"url": r.URL,
		}).Info("liveness probe OK")
		w.Write([]byte("200 OK"))
	})

	// Kubernetes readiness probe handler.
	http.HandleFunc("/k8s-ready", func(w http.ResponseWriter, r *http.Request) {
		if proxy.ready {
			log.WithFields(log.Fields{
				"url": r.URL,
			}).Info("readiness probe OK")
			w.Write([]byte("OK"))
			return
		}
		log.WithFields(log.Fields{
			"url": r.URL,
		}).Warn("503 Service Unavailable - readiness probe failed")
		w.WriteHeader(http.StatusServiceUnavailable)
		HTTPErrs[http.StatusServiceUnavailable].Execute(w, struct {
			Reason string
			Host   string
			Msg    string
		}{
			Reason: "readiness probe failed",
			Host:   r.Host,
			Msg:    "proxy service is not yet ready",
		})
	})

	// Use the passthrough handler for all other routes.
	log.WithFields(log.Fields{
		"port": proxy.Port,
	}).Info("starting kubernetes proxy")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if proxy.ready {
			proxy.Pass(w, r)
			return
		}

		// Return a 503 if the proxy service isn't ready yet.
		log.WithFields(log.Fields{
			"url": r.URL,
		}).Warn("503 Service Unavailable - Request failed, proxy not yet ready")
		w.WriteHeader(http.StatusServiceUnavailable)
		HTTPErrs[http.StatusServiceUnavailable].Execute(w, struct {
			Reason string
			Host   string
			Msg    string
		}{
			Reason: "503 Service Unavailable",
			Host:   r.Host,
			Msg:    "The proxy service is not yet ready.",
		})
	})

	// Start the HTTP passthrough server.
	go func() {
		log.WithFields(log.Fields{
			"port": proxy.Port,
		}).Info("starting HTTP passthrough service")
		errs <- http.ListenAndServe(
			fmt.Sprintf(":%d", proxy.Port),
			nil,
		)
	}()

	// Start the SSL passthrough server.
	go func() {
		log.WithFields(log.Fields{
			"cert": proxy.SSLCert,
			"port": proxy.SSLPort,
		}).Info("starting SSL passthrough service")
		errs <- http.ListenAndServeTLS(
			fmt.Sprintf(":%d", proxy.SSLPort),
			"/go/src/github.com/mkenney/k8s-proxy/assets/"+proxy.SSLCert+".crt",
			"/go/src/github.com/mkenney/k8s-proxy/assets/"+proxy.SSLCert+".key",
			nil,
		)
	}()

	proxy.readyCh <- struct{}{}
	close(proxy.readyCh)

	return errs
}

/*
Stop causes the proxy to shutdown. In a kubernetes cluster this will
cause the container to be restarted.
*/
func (proxy *Proxy) Stop() {
	proxy.K8s.Services.Stop()
}

/*
UpdateServices processes changes to the set of available services in the
cluster.
*/
func (proxy *Proxy) UpdateServices(delta k8s.ChangeSet) {
	for _, service := range delta.Added {
		proxy.AddService(service)
	}
	for _, service := range delta.Removed {
		proxy.RemoveService(service)
	}
}

/*
Wait will block until the k8s services are ready
*/
func (proxy *Proxy) Wait() {
	if proxy.ready {
		return
	}
	<-proxy.readyCh
	proxy.ready = true
}
