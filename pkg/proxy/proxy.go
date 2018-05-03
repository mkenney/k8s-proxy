package proxy

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/mkenney/k8s-proxy/pkg/k8s"
	log "github.com/sirupsen/logrus"
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
	sslPort int,
	timeout int,
) (*Proxy, error) {
	var err error
	proxy := &Proxy{
		Port:    port,
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
	Name     string
	Port     int32
	Protocol string
	Proxy    *ReverseProxy
}

/*
AddService adds a service to the passthrough map.
*/
func (proxy *Proxy) AddService(service apiv1.Service) error {
	if service.Name == "k8s-proxy" {
		return fmt.Errorf("'k8s-proxy' cannot be a proxy target")
	}
	for _, port := range service.Spec.Ports {
		if port.Port >= 80 && "TCP" == port.Protocol {

			domain := service.Name
			if _, ok := service.Labels["k8s-proxy-domain"]; ok {
				domain = service.Labels["k8s-proxy-domain"]
			}

			rp, err := NewReverseProxy(service, port)
			if nil != err {
				return err
			}

			log.WithFields(log.Fields{
				"port":    port.Port,
				"service": service.Name,
				"url":     fmt.Sprintf("//%s.*", domain),
			}).Info("registering service")

			proxy.svcMapMux.Lock()
			proxy.serviceMap[domain] = &Service{
				Name:     service.Name,
				Port:     port.Port,
				Protocol: string(port.Protocol),
				Proxy:    rp,
			}
			proxy.svcMapMux.Unlock()
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
	for k, service := range proxy.serviceMap {
		if strings.HasPrefix(r.Host, k+".") {
			return service, nil
		}
	}
	return nil, fmt.Errorf("service not found")
}

var faviconBytes []byte

/*
getFaviconBytes returns the favicon.ico data.
*/
func getFaviconBytes() []byte {
	var err error
	if nil == faviconBytes || 0 == len(faviconBytes) {
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
	svc, err := proxy.getService(r)
	protocol := "http"
	if nil != r.TLS {
		protocol = "https"
	}

	if nil != err {
		log.WithFields(log.Fields{
			"url": fmt.Sprintf("%s://%s%s", protocol, r.Host, r.URL),
		}).Warn("request failed, no matching service found")

		if "/favicon.ico" == r.URL.String() {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "image/vnd.microsoft.icon")
			w.Write(getFaviconBytes())

		} else {
			w.WriteHeader(http.StatusBadGateway)
			HTTPErrs[http.StatusBadGateway].Execute(w, struct {
				Host     string
				Scheme   string
				Services map[string]*Service
			}{
				Host:     r.Host,
				Scheme:   strings.ToUpper(protocol),
				Services: proxy.serviceMap,
			})
		}
		return
	}

	log.WithFields(log.Fields{
		"endpoint": svc.Proxy.URL,
		"referer":  r.Referer(),
	}).Infof("serving request")

	// Inject our own ResponseWriter to intercept the result of the
	// proxied request.
	proxyWriter := &ResponseWriter{200, make([]byte, 0), http.Header{}}
	svc.Proxy.ServeHTTP(proxyWriter, r)

	// Write headers first.
	for k, v := range proxyWriter.Header() {
		w.Header().Set(k, v[0])
	}
	if 502 == proxyWriter.Status() {
		log.WithFields(log.Fields{
			"status": http.StatusText(proxyWriter.Status()),
			"host":   r.Host,
		}).Infof("service responded with an error")

		if "/favicon.ico" == r.URL.String() {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "image/vnd.microsoft.icon")
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
	for _, port := range service.Spec.Ports {
		domain := service.Name
		if _, ok := service.Labels["k8s-proxy-domain"]; ok {
			domain = service.Labels["k8s-proxy-domain"]
		}

		if _, ok := proxy.serviceMap[domain]; ok {
			log.WithFields(log.Fields{
				"name": service.Name,
				"port": port.Port,
			}).Info("removing service")
			proxy.svcMapMux.Lock()
			delete(proxy.serviceMap, domain)
			proxy.svcMapMux.Unlock()
		}
	}

	return nil
}

/*
Start starts the proxy.
*/
func (proxy *Proxy) Start() chan error {
	errs := make(chan error)

	// Set the global timeout.
	http.DefaultClient.Timeout = time.Duration(proxy.Timeout) * time.Second

	// Start the change watcher and the updater. This will block until
	// data is available.
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
		}).Infof("liveness probe OK")
		w.Write([]byte("200 OK"))
	})

	// Kubernetes readiness probe handler.
	http.HandleFunc("/k8s-ready", func(w http.ResponseWriter, r *http.Request) {
		if proxy.ready {
			log.WithFields(log.Fields{
				"url": r.URL,
			}).Infof("readiness probe OK")
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
			"port": proxy.SSLPort,
		}).Infof("starting SSL passthrough service")
		errs <- http.ListenAndServeTLS(
			fmt.Sprintf(":%d", proxy.SSLPort),
			"/go/src/github.com/mkenney/k8s-proxy/assets/k8s-proxy.crt",
			"/go/src/github.com/mkenney/k8s-proxy/assets/k8s-proxy.key",
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
