package proxy

import (
	"fmt"
	"net"
	"strings"
	"time"

	errs "github.com/bdlm/errors"
	apiv1 "k8s.io/api/core/v1"
)

// NewConn creates a new reverse proxy to forward traffic through.
func NewConn(protocol string, host string, port int32, service apiv1.Service) *Conn {
	return &Conn{
		address:  strings.ToLower(fmt.Sprintf("%s:%d", host, port)),
		host:     strings.ToLower(host),
		port:     port,
		protocol: strings.ToLower(protocol),
		service:  service,
	}
}

// Conn defines a proxy to a service.
type Conn struct {
	address  string
	host     string
	port     int32
	protocol string
	service  apiv1.Service
}

// Address returns the target network address for this connection.
func (conn *Conn) Address() string {
	return conn.address
}

// Host returns the target network host for this connection.
func (conn *Conn) Host() string {
	return conn.host
}

// Pass forwards network requests to a service and returns the response.
func (conn *Conn) Pass(request net.Conn) error {
	dialer := &net.Dialer{
		Deadline:      time.Now().Add(30 * time.Second),
		LocalAddr:     request.RemoteAddr(),
		DualStack:     true,
		FallbackDelay: 50 * time.Millisecond,
	}

	service, err := dialer.Dial(conn.Protocol(), conn.Address())
	if nil != err {
		return err
	}
	defer service.Close()

	b := []byte{}
	r1, err := request.Read(b)
	if nil != err {
		return errs.Wrap(err, 0, "error reading request")
	}

	s1, err := service.Write(b)
	if nil != err {
		return errs.Wrap(err, 0, "error writing request to service")
	}

	b = []byte{}
	s2, err := service.Read(b)
	if nil != err {
		return errs.Wrap(err, 0, "error reading service response")
	}

	r2, err := request.Write(b)
	if nil != err {
		return errs.Wrap(err, 0, "error writing service response to request")
	}

	if r1 != s1 || r2 != s2 {
		return fmt.Errorf("request bytes read: %d, service bytes written: %d, service bytes read: %d, request bytes written: %d. all values should be equal but are not", r1, s1, s2, r2)
	}

	return nil
}

// Port returns the target port for this connection..
func (conn *Conn) Port() int32 {
	return conn.port
}

// Protocol returns the network protocol for this connection.
func (conn *Conn) Protocol() string {
	return conn.protocol
}

// String returns the target network type and host of the dialed service.
func (conn *Conn) String() string {
	return fmt.Sprintf(
		"%s %s:%d",
		conn.Protocol(),
		conn.Host(),
		conn.Port(),
	)
}
