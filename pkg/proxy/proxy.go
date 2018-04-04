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
ServiceMap is a map of k8s service name to proxy description.
*/
type ServiceMap map[string]map[string]*Service

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

	svcMapMux  sync.Mutex
	ServiceMap ServiceMap
}

/*
New initializes the proxy service and  returns a pointer to the service
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
		ServiceMap: ServiceMap{
			"http":  make(map[string]*Service),
			"https": make(map[string]*Service),
		},
		Timeout: timeout,
	}
	proxy.K8s, err = k8s.New()
	return proxy, err
}

/*
AddService adds a service to the map.
*/
func (proxy *Proxy) AddService(service apiv1.Service) error {
	proxy.svcMapMux.Lock()
	defer proxy.svcMapMux.Unlock()

	for _, port := range service.Spec.Ports {
		if "TCP" == port.Protocol && service.Name != "k8s-proxy" {
			log.Debugf("registering service '%s:%d'", service.Name, port.Port)
			rp, err := NewReverseProxy(service)
			if nil != err {
				return err
			}

			scheme := "http"
			if "443" == rp.URL.Port() {
				scheme = "https"
			}

			proxy.ServiceMap[scheme][service.Name] = &Service{
				Name:     service.Name,
				Port:     port.Port,
				Protocol: string(port.Protocol),
				Proxy:    rp,
			}
		}
	}
	return nil
}

/*
RemoveService removes a service from the map.
*/
func (proxy *Proxy) RemoveService(service apiv1.Service) error {
	proxy.svcMapMux.Lock()
	defer proxy.svcMapMux.Unlock()

	log.Debugf("removing service '%s'", service.Name)
	for _, port := range service.Spec.Ports {
		scheme := "http"
		if 443 == port.Port {
			scheme = "https"
		}
		if _, ok := proxy.ServiceMap[scheme][service.Name]; ok {
			return fmt.Errorf("removed service '%s'", service.Name)
		}
	}

	delete(proxy.ServiceMap, service.Name)

	return nil
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

func (proxy *Proxy) hostToService(host string) string {
	for _, scheme := range []string{"http", "https"} {
		for k := range proxy.ServiceMap[scheme] {
			if strings.HasPrefix(host, k+".") {
				return k
			}
		}
	}
	return proxy.Default
}

/*
Pass passes HTTP traffic through to the requested service.
*/
func (proxy *Proxy) Pass(w http.ResponseWriter, r *http.Request) {
	scheme := "http"
	if nil != r.TLS {
		scheme = "https"
	}
	service := proxy.hostToService(r.Host)

	log.Infof("new request: scheme=%s, request=%s, ip=%s, service=%s", scheme, r.Host, r.RemoteAddr, service)
	if svc, ok := proxy.ServiceMap[scheme][service]; ok {
		log.Debugf("serving %s://%s:%d%s", scheme, service, proxy.Port, r.URL.Path)
		svc.Proxy.proxy.ServeHTTP(w, r)
	} else {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(fmt.Sprintf(HTTPErrs[502], strings.ToUpper(scheme), service)))
	}
}

/*
Start starts the proxy.
*/
func (proxy *Proxy) Start() chan error {
	errs := make(chan error)

	// Set the global timeout.
	http.DefaultClient.Timeout = time.Duration(proxy.Timeout) * time.Second

	// Start the change watcher and the updater.
	changes := proxy.K8s.Services.Watch()
	go func() {
		for delta := range changes {
			proxy.UpdateServices(delta)
			time.Sleep(5 * time.Second)
		}
	}()

	// Add kubernetes healthcheck endpoints.
	http.HandleFunc("/alive", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("alive"))
	})
	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ready"))
	})

	// Add passthrough handler.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		proxy.Pass(w, r)
	})

	log.WithFields(log.Fields{
		"port": proxy.Port,
	}).Info("starting kubernetes proxy")

	go func() {
		log.Infof("starting unsecured proxy on port %d", proxy.Port)
		errs <- http.ListenAndServe(fmt.Sprintf(":%d", proxy.Port), nil)
	}()
	//go func() {
	//	log.Infof("starting secured proxy on port %d", proxy.SecurePort)
	//	errs <- http.ListenAndServe(fmt.Sprintf(":%d", proxy.SecurePort), nil)
	//}()
	go func() {
		log.Infof("starting secured proxy on port %d", proxy.SecurePort)
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
cause the service to restart.
*/
func (proxy *Proxy) Stop() {
	proxy.K8s.Services.Stop()
}
