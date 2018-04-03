package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/mkenney/k8s-proxy/src/k8s"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	//"rsc.io/letsencrypt"
)

type ServiceMap map[string]*Service

type Service struct {
	Name     string
	Port     int32
	Protocol string
	Proxy    *ReverseProxy
	URL      *url.URL
}

func (s Service) String() string {
	return fmt.Sprintf(
		"%s://%s:%d",
		s.Protocol,
		s.Name,
		s.Port,
	)
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
	Secure     bool
	ServiceMap ServiceMap
	Timeout    int
}

/*
New initializes the proxy service and  returns a pointer to the service
instance. If an error is generated while initializing the kubernetes
service scanner an error will be returned.
*/
func New() (*Proxy, error) {
	var err error
	proxy := &Proxy{
		Default:    "kubernetes-dashboard",
		Dev:        true,
		Port:       80,
		Secure:     false,
		ServiceMap: make(ServiceMap),
		Timeout:    60,
	}
	proxy.K8s, err = k8s.New()
	return proxy, err
}

// Store all of our proxies
var endpoints map[string]*ReverseProxy
var endpointkeys sort.StringSlice

/*
AddService adds a service to the map.
*/
func (proxy *Proxy) AddService(host string, service apiv1.Service) error {
	log.Debugf("adding service %s: %s", service.Name, host)
	for _, port := range service.Spec.Ports {
		if "TCP" == port.Protocol {
			uri, err := url.Parse(host)
			if nil != err {
				return err
			}
			rp, err := NewReverseProxy(service)
			if nil != err {
				return err
			}
			proxy.ServiceMap[service.Name] = &Service{
				Name:     service.Name,
				URL:      uri,
				Port:     port.Port,
				Protocol: string(port.Protocol),
				Proxy:    rp,
			}
			break
		}
	}
	return nil
}

/*
RemoveService removes a service from the map.
*/
func (proxy *Proxy) RemoveService(host string, service apiv1.Service) error {
	log.Debugf("removing service %s: %s", service.Name, host)
	if _, ok := proxy.ServiceMap[service.Name]; !ok {
		return fmt.Errorf("service not registered '%s'", service.Name)
	}
	delete(proxy.ServiceMap, service.Name)
	return nil
}

/*
UpdateServices processes changes to the set of available services in the
cluster.
*/
func (proxy *Proxy) UpdateServices(delta k8s.ChangeSet) {
	var changed bool
	for host, service := range delta.Added {
		proxy.AddService(host, service)
		changed = true
	}
	for host, service := range delta.Removed {
		proxy.RemoveService(host, service)
		changed = true
	}
	if changed {
		// trigger reload of proxies...?
	}
}

func (proxy *Proxy) hostToService(host string) string {
	for k := range proxy.ServiceMap {
		if strings.HasPrefix(host, k) {
			return k
		}
	}
	return proxy.Default
}

/*
Pass passes HTTP traffic through to the requested service.
*/
func (proxy *Proxy) Pass(w http.ResponseWriter, r *http.Request) {
	service := proxy.hostToService(r.Host)

	log.Infof("new request: request=%s, ip=%s, service=%s", r.Host, r.RemoteAddr, service)

	// One quick sanity check before sending it on it's way
	if _, ok := proxy.ServiceMap[service]; ok {
		proxy.ServiceMap[service].Proxy.proxy.ServeHTTP(w, r)
	} else {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Error 502 - Bad Gateway"))
	}
}

/*
Start starts the proxy.
*/
func (proxy *Proxy) Start() error {
	// Set the global timeout.
	http.DefaultClient.Timeout = time.Duration(proxy.Timeout) * time.Second

	// Start the change watcher and the updater.
	changes := proxy.K8s.Services.Watch()

	tmp, _ := json.MarshalIndent(changes, "", "    ")
	log.Debugf("'%s'", string(tmp))

	go func() {
		for delta := range changes {
			time.Sleep(5 * time.Second)
			proxy.UpdateServices(delta)
		}
	}()

	// Initialize the service map
	for host, service := range proxy.K8s.Services.Map() {
		proxy.AddService(host, service)
	}

	// Add healthcheck endpoints
	http.HandleFunc("/alive", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte{})
	})
	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte{})
	})

	// Add passthrough handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		proxy.Pass(w, r)
	})

	log.WithFields(log.Fields{
		"port": proxy.Port,
	}).Info("starting kubernetes proxy")

	// Not secured, just start a basic webserver
	if !proxy.Secure {
		go func() {
			log.Debugf("starting unsecured proxy")
			log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", proxy.Port), nil))
		}()
	} else {
		// start our letsencrypt SSL goodies
		//var m letsencrypt.Manager
		//if err := m.CacheFile("letsencrypt.cache"); err != nil {
		//	log.Fatal(err)
		//}
		//log.Fatal(m.Serve())
	}
	return nil
}

/*
AddSite adds a new website to the proxy to be forwarded.
*/
//func (proxy *Proxy) AddSite(
//	base string,
//	address *url.URL,
//	healthChecks bool,
//	healthCheckURL string,
//) error {
//	// Check if endpoint already exists
//	for _, item := range endpoints {
//		if item.Registered == base && item.Address.String() == address.String() {
//			return nil
//		}
//	}
//
//	// Construct the key so that you can sort by url base and time added
//	urlbase := base
//
//	// Remove any thing after the _ from the url
//	if strings.Contains(urlbase, "_") {
//		urlbase = urlbase[0:strings.Index(urlbase, "_")]
//	}
//	key := urlbase + "-" + time.Now().Format("2006-01-02T15:04:05.000")
//
//	// Add new endpoint
//	rp, err := NewReverseProxy(base, address, healthChecks, healthCheckURL)
//	if err == nil {
//		// If it doesn't exist ...
//		log.WithFields(log.Fields{
//			"url":        address,
//			"registered": base,
//			"urlbase":    urlbase,
//		}).Info("Registered endpoint")
//		endpoints[key] = rp
//		endpointkeys = append(endpointkeys, key)
//
//		sort.Sort(sort.Reverse(endpointkeys))
//
//		return nil
//	}
//	return err
//}

// HealthChecks starts the background process for __all__ site health checks
func HealthChecks(healthCheckURL string) {
	for {
		<-time.After(10 * time.Second)
		for key := range endpoints {
			go endpoints[key].HealthCheck(healthCheckURL)
		}
	}
}
