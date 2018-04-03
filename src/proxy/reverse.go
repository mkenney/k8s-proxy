package proxy

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
)

/*
NewReverseProxy creates a new reverse proxy to forward traffic through.
*/
func NewReverseProxy(
	service apiv1.Service,
) (*ReverseProxy, error) {
	//base string,
	//addr *url.URL,
	//checkHealth bool,
	//healthURL string,

	proto := "http"
	if 443 == service.Spec.Ports[0].Port {
		proto = "https"
	}

	port := service.Spec.Ports[0].Port
	if 0 > service.Spec.Ports[0].NodePort {
		port = service.Spec.Ports[0].NodePort
	}
	host := fmt.Sprintf(
		"%s://%s.%s.svc.cluster.local:%d",
		proto,
		service.Name,
		service.Namespace,
		//service.Spec.Ports[0].Port,
		port,
	)
	URL, err := url.Parse(host)
	if nil != err {
		return nil, err
	}

	rp := &ReverseProxy{
		URL:       URL,
		proxy:     httputil.NewSingleHostReverseProxy(URL),
		Active:    true,
		Available: time.Now(),
		Service:   service.Name,
	}
	rp.proxy.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	}

	//	if checkHealth {
	//		rp.HealthCheck(healthURL)
	//	} else {
	//		rp.Active = true
	//	}
	return rp, nil
}

/*
ReverseProxy defines a proxy to a service.
*/
type ReverseProxy struct {
	Active    bool
	Available time.Time
	Service   string
	URL       *url.URL

	proxy *httputil.ReverseProxy
}

func (rp *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rp.proxy.ServeHTTP(w, r)
}

// HealthCheck performs a basic http check based on a positive(<500) status code
func (rp *ReverseProxy) HealthCheck(healthCheckURL string) {
	previousStatus := rp.Active
	statusCode := 500
	if resp, err := http.Get(rp.URL.String() + "/" + healthCheckURL); err != nil {
		// Something is up ... disable this reverse proxy
		rp.Active = false
	} else {
		// Woot! Good to go ...
		statusCode = resp.StatusCode
		if resp.StatusCode >= 500 {
			rp.Active = false
		} else {
			rp.Active = true
		}
	}
	log.WithFields(log.Fields{
		"previous": previousStatus,
		"current":  rp.Active,
	}).Debug(rp.Service)
	if rp.Active != previousStatus {
		if rp.Active {
			// Whew, we came back online
			log.WithFields(
				log.Fields{
					"URL": rp.URL.String(),
				}).Info("Up")
			rp.Available = time.Now()
		} else {
			// BOO HISS!
			log.WithFields(
				log.Fields{
					"URL":    rp.URL.String(),
					"Status": statusCode,
				}).Error("Down")
		}
	}
}
