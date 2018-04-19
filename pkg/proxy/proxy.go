package proxy

import (
	"fmt"
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
	defaultService string,
	dev bool,
	port int,
	securePort int,
	timeout int,
) (*Proxy, error) {
	var err error
	proxy := &Proxy{
		Default:    defaultService,
		Dev:        dev,
		Port:       port,
		SecurePort: securePort,
		Timeout:    timeout,

		readyCh: make(chan struct{}, 2),
		serviceMap: ServiceMap{
			"http":  make(map[string]*Service),
			"https": make(map[string]*Service),
		},
	}
	proxy.K8s, err = k8s.New()
	return proxy, err
}

/*
Proxy holds configuration data and methods for running the kubernetes
proxy service.
*/
type Proxy struct {
	Default    string
	Dev        bool
	K8s        *k8s.K8S
	Port       int
	SecurePort int
	Timeout    int

	ready      bool
	readyCh    chan struct{}
	svcMapMux  sync.Mutex
	serviceMap ServiceMap
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
ServiceMap is a map of k8s service name to proxy description.
*/
type ServiceMap map[string]map[string]*Service

/*
AddService adds a service to the passthrough map.
*/
func (proxy *Proxy) AddService(service apiv1.Service) error {
	if service.Name == "k8s-proxy" {
		return fmt.Errorf("'k8s-proxy' cannot be a proxy target")
	}
	for _, port := range service.Spec.Ports {
		if port.Port >= 80 && "TCP" == port.Protocol {

			protocol := "http"
			if 443 == port.Port {
				protocol = "https"
			}
			if _, ok := service.Labels["k8s-proxy-protocol"]; ok {
				protocol = strings.ToLower(service.Labels["k8s-proxy-protocol"])
			}

			domain := service.Name
			if _, ok := service.Labels["k8s-proxy-domain"]; ok {
				domain = service.Labels["k8s-proxy-domain"]
			}

			rp, err := NewReverseProxy(service, port)
			if nil != err {
				return err
			}

			log.WithFields(log.Fields{
				"name":     domain,
				"port":     port.Port,
				"protocol": protocol,
			}).Info("registering service")

			proxy.svcMapMux.Lock()
			proxy.serviceMap[protocol][domain] = &Service{
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
Pass passes HTTP traffic through to the requested service.
*/
func (proxy *Proxy) Pass(w http.ResponseWriter, r *http.Request) {
	protocol := "http"
	if nil != r.TLS {
		protocol = "https"
	}

	service := proxy.Default
	for k := range proxy.serviceMap[protocol] {
		if strings.HasPrefix(r.Host, k+".") {
			service = k
			break
		}
	}

	if svc, ok := proxy.serviceMap[protocol][service]; ok {
		log.WithFields(log.Fields{
			"endpoint": svc.Proxy.URL,
			"referer":  r.Referer(),
		}).Infof("serving request")

		// wrap it to capture the status code
		proxyWriter := &ResponseWriter{200, w}
		svc.Proxy.ServeHTTP(proxyWriter, r)

		if 502 == proxyWriter.Status() {
			log.WithFields(log.Fields{
				"status": http.StatusText(proxyWriter.Status()),
				"host":   r.Host,
			}).Infof("service responded with an error")
			HTTPErrs[503].Execute(w, struct {
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
		log.WithFields(log.Fields{
			"url": fmt.Sprintf("%s://%s%s", r.Proto, r.Host, r.URL),
		}).Warn("request failed, no matching service found")
		w.WriteHeader(http.StatusBadGateway)
		HTTPErrs[502].Execute(w, struct {
			Host     string
			Scheme   string
			Services map[string]*Service
		}{
			Host:     r.Host,
			Scheme:   strings.ToUpper(protocol),
			Services: proxy.serviceMap[protocol],
		})
	}
}

/*
RemoveService removes a service from the map.
*/
func (proxy *Proxy) RemoveService(service apiv1.Service) error {
	for _, port := range service.Spec.Ports {
		protocol := "http"
		if 443 == port.Port {
			protocol = "https"
		}
		if _, ok := service.Labels["k8s-proxy-protocol"]; ok {
			protocol = service.Labels["k8s-proxy-protocol"]
		}

		domain := service.Name
		if _, ok := service.Labels["k8s-proxy-domain"]; ok {
			domain = service.Labels["k8s-proxy-domain"]
		}

		log.WithFields(log.Fields{
			"name":     domain,
			"port":     port.Port,
			"protocol": protocol,
		}).Info("removing service")

		if _, ok := proxy.serviceMap[protocol][domain]; ok {
			proxy.svcMapMux.Lock()
			delete(proxy.serviceMap[protocol], domain)
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
	changes := proxy.K8s.Services.Watch(5)
	proxy.readyCh <- struct{}{}
	close(proxy.readyCh)
	go func() {
		for delta := range changes {
			proxy.UpdateServices(delta)
			time.Sleep(5 * time.Second)
		}
	}()

	// Kubernetes liveness probe handler.
	http.HandleFunc("/x8s-alive", func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(log.Fields{
			"url": r.URL,
		}).Infof("liveness probe OK")
		w.Write([]byte("200 OK"))
	})

	// Kubernetes readiness probe handler.
	http.HandleFunc("/x8s-ready", func(w http.ResponseWriter, r *http.Request) {
		if proxy.ready {
			log.WithFields(log.Fields{
				"url": r.URL,
			}).Infof("readiness probe OK")
			w.Write([]byte("OK"))
			return
		}
		log.WithFields(log.Fields{
			"url": r.URL,
		}).Error("Service Unavailable - readiness probe failed")
		w.WriteHeader(http.StatusServiceUnavailable)
		HTTPErrs[503].Execute(w, struct {
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
		proxy.Pass(w, r)
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
			"port": proxy.SecurePort,
		}).Infof("starting SSL passthrough service")
		errs <- http.ListenAndServeTLS(
			fmt.Sprintf(":%d", proxy.SecurePort),
			"/go/src/github.com/mkenney/k8s-proxy/server.crt",
			"/go/src/github.com/mkenney/k8s-proxy/server.key",
			nil,
		)
	}()

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

/*
hostToService takes in a host string (eg. service.example.com) and
matches it to a running kubernetes service.
*/
func (proxy *Proxy) hostToService(host string) string {
	for _, scheme := range []string{"http", "https"} {
		for k := range proxy.serviceMap[scheme] {
			if strings.HasPrefix(host, k+".") {
				return k
			}
		}
	}
	return proxy.Default
}
