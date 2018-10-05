package proxy

import (
	"net"
)

type Request struct {
	Conn net.Conn
	Err  error
}
