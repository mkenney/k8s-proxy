package proxy

import (
	"net"
)

// Request represents a network request to be passed to a service.
type Request struct {
	Conn net.Conn
	Err  error
}
