package proxy

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	apiv1 "k8s.io/api/core/v1"
)

/*
NewReverseProxy creates a new reverse proxy to forward traffic through.
*/
func NewReverseProxy(service apiv1.Service, port apiv1.ServicePort) (*ReverseProxy, error) {
	protocol := "http"
	if 443 == port.Port {
		protocol = "https"
	}
	if "https" == port.Protocol {
		protocol = "https"
	}
	if _, ok := service.Labels["k8s-proxy-protocol"]; ok {
		protocol = service.Labels["k8s-proxy-protocol"]
	}

	proxyPort := port.Port
	if 0 > port.NodePort {
		proxyPort = service.Spec.Ports[0].NodePort
	}

	clusterURL, err := url.Parse(fmt.Sprintf(
		"%s://%s.%s.svc.cluster.local:%d",
		protocol,
		service.Name,
		service.Namespace,
		proxyPort,
	))
	if nil != err {
		return nil, err
	}

	rp := &ReverseProxy{
		URL:       clusterURL,
		proxy:     httputil.NewSingleHostReverseProxy(clusterURL),
		Active:    true,
		Available: time.Now(),
		Service:   service.Name,
	}

	// Don't validate SSL certificates
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	rp.proxy.FlushInterval = 0

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

/*
String implements stringer. Return the URL for this proxy.
*/
func (rp *ReverseProxy) String() string {
	return rp.URL.String()
}

/*
ServeHTTP starts the HTTP server for this proxy.
*/
func (rp *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rp.proxy.ServeHTTP(w, r)
}
