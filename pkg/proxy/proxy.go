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
	dev bool,
	port int,
	securePort int,
	timeout int,
) (*Proxy, error) {
	var err error
	proxy := &Proxy{
		Dev:        dev,
		Port:       port,
		SecurePort: securePort,
		Timeout:    timeout,

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
	Dev        bool
	K8s        *k8s.K8S
	Port       int
	SecurePort int
	Timeout    int

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
				"name": domain,
				"port": port.Port,
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
Pass passes HTTP traffic through to the requested service.
*/
func (proxy *Proxy) Pass(w http.ResponseWriter, r *http.Request) {
	var err error
	service := ""
	for k := range proxy.serviceMap {
		if strings.HasPrefix(r.Host, k+".") {
			service = k
			break
		}
	}

	// Find the correct service proxy and route the traffic.
	if svc, ok := proxy.serviceMap[service]; ok {
		log.WithFields(log.Fields{
			"endpoint": svc.Proxy.URL,
			"referer":  r.Referer(),
		}).Infof("serving request")

		// Wrap the ResponseWriter it to intercept the resulting status
		// code of the proxied request.
		proxyWriter := &ResponseWriter{200, make([]byte, 0), http.Header{}}
		svc.Proxy.ServeHTTP(proxyWriter, r)

		if 502 == proxyWriter.Status() {
			log.WithFields(log.Fields{
				"status": http.StatusText(proxyWriter.Status()),
				"host":   r.Host,
			}).Infof("service responded with an error")

			if "/favicon.ico" == r.URL.String() {
				if nil == faviconBytes || 0 == len(faviconBytes) {
					faviconBytes, err = ioutil.ReadFile("/go/src/github.com/mkenney/k8s-proxy/assets/favicon.ico")
					if nil != err {
						log.Error(err)
					}
				}

				w.WriteHeader(http.StatusOK)
				w.Header().Set("Content-Type", "image/vnd.microsoft.icon")
				_, err = w.Write(faviconBytes)
				if nil != err {
					log.Error(err)
				}

			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
				err = HTTPErrs[503].Execute(w, struct {
					Reason string
					Host   *url.URL
					Msg    string
				}{
					Reason: fmt.Sprintf("%d %s", proxyWriter.Status(), http.StatusText(proxyWriter.Status())),
					Host:   svc.Proxy.URL,
					Msg:    "The deployed pod(s) may be unavailable or unresponsive.",
				})
				if nil != err {
					log.Error(err)
				}
			}
		} else {
			w.WriteHeader(proxyWriter.Status())
		}

		for k, v := range proxyWriter.Header() {
			w.Header()[k] = v
		}
		w.Write(proxyWriter.data)

	} else {
		protocol := "http"
		if nil != r.TLS {
			protocol = "https"
		}
		log.WithFields(log.Fields{
			"url": fmt.Sprintf("%s://%s%s", protocol, r.Host, r.URL),
		}).Warn("request failed, no matching service found")

		if "/favicon.ico" == r.URL.String() {
			if nil == faviconBytes || 0 == len(faviconBytes) {
				faviconBytes, err = ioutil.ReadFile("/go/src/github.com/mkenney/k8s-proxy/assets/favicon.ico")
				if nil != err {
					log.Error(err)
				}
			}

			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "image/vnd.microsoft.icon")
			_, err = w.Write(faviconBytes)
			if nil != err {
				log.Error(err)
			}

		} else {
			w.WriteHeader(http.StatusBadGateway)
			err := HTTPErrs[502].Execute(w, struct {
				Host     string
				Scheme   string
				Services map[string]*Service
			}{
				Host:     r.Host,
				Scheme:   strings.ToUpper(protocol),
				Services: proxy.serviceMap,
			})
			if nil != err {
				log.Error(err)
			}
		}
	}
}

var faviconBytes []byte

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
			"/go/src/github.com/mkenney/k8s-proxy/k8s-proxy.crt",
			"/go/src/github.com/mkenney/k8s-proxy/k8s-proxy.key",
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
	for k := range proxy.serviceMap {
		if strings.HasPrefix(host, k+".") {
			return k
		}
	}
	return ""
}
