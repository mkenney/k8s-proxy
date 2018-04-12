package proxy

import (
	"fmt"
	"net/http"
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
AddService adds a service to the map.
*/
func (proxy *Proxy) AddService(service apiv1.Service) error {

	for _, port := range service.Spec.Ports {
		if "TCP" == port.Protocol && service.Name != "k8s-proxy" {
			log.WithFields(log.Fields{
				"name": service.Name,
				"port": port.Port,
			}).Info("registering service")

			rp, err := NewReverseProxy(service)
			if nil != err {
				return err
			}

			scheme := "http"
			if "443" == rp.URL.Port() {
				scheme = "https"
			}

			key := ""
			ok := false
			if key, ok = service.Labels["domain"]; !ok {
				key = service.Name
			}

			proxy.svcMapMux.Lock()
			proxy.serviceMap[scheme][key] = &Service{
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
	scheme := "http"
	if nil != r.TLS {
		scheme = "https"
	}

	service := proxy.Default
	for _, scheme := range []string{"http", "https"} {
		for k := range proxy.serviceMap[scheme] {
			if strings.HasPrefix(r.Host, k+".") {
				service = k
				break
			}
		}
	}

	if svc, ok := proxy.serviceMap[scheme][service]; ok {
		log.WithFields(log.Fields{
			"endpoint": svc.Proxy.URL,
			"request":  fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL),
		}).Infof("serving request")
		svc.Proxy.ServeHTTP(w, r)

	} else {
		log.WithFields(log.Fields{
			"url": fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL),
		}).Warn("request failed, no matching service found")
		w.WriteHeader(http.StatusBadGateway)
		HTTPErrs[502].Execute(w, struct {
			Host     string
			Scheme   string
			Services map[string]*Service
		}{
			Host:     r.Host,
			Scheme:   strings.ToUpper(scheme),
			Services: proxy.serviceMap[scheme],
		})
	}
}

/*
RemoveService removes a service from the map.
*/
func (proxy *Proxy) RemoveService(service apiv1.Service) error {
	for _, port := range service.Spec.Ports {
		scheme := "http"
		if 443 == port.Port {
			scheme = "https"
		}

		key := ""
		ok := false
		if key, ok = service.Labels["domain"]; !ok {
			key = service.Name
		}

		if _, ok := proxy.serviceMap[scheme][key]; ok {
			log.WithFields(log.Fields{
				"name": service.Name,
				"port": port.Port,
			}).Info("removing service")

			proxy.svcMapMux.Lock()
			delete(proxy.serviceMap[scheme], service.Name)
			proxy.svcMapMux.Unlock()
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

	// Start the change watcher and the updater. This will block until data is available.
	changes := proxy.K8s.Services.Watch(5)
	proxy.readyCh <- struct{}{}
	close(proxy.readyCh)
	go func() {
		for delta := range changes {
			proxy.UpdateServices(delta)
			time.Sleep(5 * time.Second)
		}
	}()

	// Kubernetes liveness probe.
	http.HandleFunc("/xalive", func(w http.ResponseWriter, r *http.Request) {
		log.Debug("liveness probe OK")
		w.Write([]byte("OK"))
	})

	// Kubernetes readiness probe.
	http.HandleFunc("/xready", func(w http.ResponseWriter, r *http.Request) {
		if proxy.ready {
			log.Debug("readiness probe OK")
			w.Write([]byte("OK"))
			return
		}

		log.Error("readiness probe failed")
		w.WriteHeader(http.StatusServiceUnavailable)
		HTTPErrs[503].Execute(w, struct {
			Reason string
			Host   string
		}{
			Reason: "readiness probe failed",
			Host:   r.Host,
		})
	})

	// Add passthrough handler.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		proxy.Pass(w, r)
	})

	log.WithFields(log.Fields{
		"port": proxy.Port,
	}).Info("starting kubernetes proxy")

	go func() {
		log.WithFields(log.Fields{
			"port": proxy.Port,
		}).Info("starting HTTP passthrough service")
		errs <- http.ListenAndServe(
			fmt.Sprintf(":%d", proxy.Port),
			nil,
		)
	}()
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
