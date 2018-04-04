package proxy

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	apiv1 "k8s.io/api/core/v1"
)

/*
NewReverseProxy creates a new reverse proxy to forward traffic through.
*/
func NewReverseProxy(
	service apiv1.Service,
) (*ReverseProxy, error) {

	scheme := "http"
	if 443 == service.Spec.Ports[0].Port {
		scheme = "https"
	}

	port := service.Spec.Ports[0].Port
	if 0 > service.Spec.Ports[0].NodePort {
		port = service.Spec.Ports[0].NodePort
	}

	URL, err := url.Parse(fmt.Sprintf(
		"%s://%s.%s.svc.cluster.local:%d",
		scheme,
		service.Name,
		service.Namespace,
		port,
	))
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

	// Don't validate SSL certificates
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	//rp.proxy.Transport = &http.Transport{
	//	Proxy: http.ProxyFromEnvironment,
	//	Dial: (&net.Dialer{
	//		Timeout:   30 * time.Second,
	//		KeepAlive: 30 * time.Second,
	//	}).Dial,
	//	TLSHandshakeTimeout: 10 * time.Second,
	//	TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	//}
	//rp.proxy.Transport = &ConnectionErrorHandler{http.DefaultTransport}
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

func (rp *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rp.proxy.ServeHTTP(w, r)
}

type ConnectionErrorHandler struct{ http.RoundTripper }

func (c *ConnectionErrorHandler) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := c.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if _, ok := err.(*net.OpError); ok {
		r := &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Body:       ioutil.NopCloser(bytes.NewBufferString(HTTPErrs[503])),
		}
		return r, nil
	}
	return resp, err
}
