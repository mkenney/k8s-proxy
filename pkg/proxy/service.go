package proxy

import (
	"fmt"
	"strings"
	"sync"

	"github.com/mkenney/k8s-proxy/pkg/k8s"
	apiv1 "k8s.io/api/core/v1"
)

// NewService creates a new reverse proxy to forward traffic through.
func NewService(model apiv1.Service, api *k8s.K8S) *Service {
	svc := &Service{
		api:   api,
		conns: make(map[string]*Conn),
		model: model,
		host:  strings.ToLower(model.Name + "." + model.Namespace),
	}

	// Create all necessary connections.
	for _, servicePort := range svc.model.Spec.Ports {
		// Protocol.
		protocol := "tcp"
		if "" != servicePort.Protocol {
			protocol = strings.ToLower(string(servicePort.Protocol))
		}

		// Port for receiving traffic.
		port := servicePort.Port
		if 0 > servicePort.TargetPort.IntVal {
			port = servicePort.TargetPort.IntVal
		}
		if 0 > servicePort.NodePort {
			port = servicePort.NodePort
		}

		// Add a connection for this port
		svc.conns[fmt.Sprintf("%s:%d", protocol, port)] = NewConn(
			protocol,
			svc.host,
			port,
			svc.model,
		)
	}

	return svc
}

// Service defines a k8s service proxy.
type Service struct {
	api   *k8s.K8S
	model apiv1.Service
	host  string

	mux   sync.Mutex
	conns map[string]*Conn
}

// Close all network connections for this service.
func (svc *Service) Close() error {
	for _, conn := range svc.Conns() {
		conn.Close()
	}
	return nil
}

// Conns returns the network connection map
func (svc *Service) Conns() map[string]*Conn {
	return svc.conns
}

// Host returns this service's hostname
func (svc *Service) Host() string {
	return svc.host
}

// Model returns the k8s service model.
func (svc *Service) Model() apiv1.Service {
	return svc.model
}

// Name returns the k8s service name.
func (svc *Service) Name() string {
	return svc.model.Name
}

// Pass passes traffic through to the requested service.
func (svc *Service) Pass(b []byte) (int, error) {
	return 0, nil
}

// Refresh checks the service deployment for updates, add any new
// connections, reconnect any disconnected connections.
func (svc *Service) Refresh() error {
	return nil
}
