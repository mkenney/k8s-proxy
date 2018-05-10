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

func init() {
	// Don't validate SSL certificates
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
}

/*
NewReverseProxy creates a new reverse proxy to forward traffic through.
*/
func NewReverseProxy(scheme string, service apiv1.Service, port int32) (*ReverseProxy, error) {

	clusterURL, err := url.Parse(fmt.Sprintf(
		"%s://%s.%s:%d",
		scheme,
		service.Name,
		service.Namespace,
		port,
	))
	if nil != err {
		return nil, err
	}

	rp := &ReverseProxy{
		Active:    true,
		Available: time.Now(),
		Service:   service.Name,
		URL:       clusterURL,
		proxy:     httputil.NewSingleHostReverseProxy(clusterURL),
	}
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
