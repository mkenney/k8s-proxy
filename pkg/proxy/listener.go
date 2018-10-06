package proxy

import (
	"context"
	"fmt"
	"net"

	errs "github.com/bdlm/errors"
	"github.com/bdlm/log"
)

// ListenerStatus defines the state of the listener.
type ListenerStatus int

const (
	// Ready represents the initial ready state.
	Ready ListenerStatus = iota
	// Listening represents the active state.
	Listening
	// Closed represents the closed state.
	Closed
)

//var netConfig = &net.Config{InsecureSkipVerify: true}

// NewListener creates a new reverse proxy to forward traffic through.
func NewListener(protocol string, port int32, requestCh chan Request) *Listener {
	return &Listener{
		closeCh:   make(chan error),
		protocol:  protocol,
		port:      port,
		requestCh: requestCh,
		status:    Ready,
	}
}

// Listener defines the network listener for services using a particular
// protocol and port.
type Listener struct {
	closeCh   chan error
	conn      net.Listener
	protocol  string
	port      int32
	requestCh chan Request
	status    ListenerStatus
}

// Listen starts the network listener.
func (l *Listener) Listen(ctx context.Context) error {
	var err error

	if Listening == l.status {
		return errs.New(0, "listener is already running")
	}
	l.conn, err = net.Listen(l.Protocol(), fmt.Sprintf(":%d", l.Port()))
	l.status = Listening

	// Start the listen loop
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn, err := l.conn.Accept()
				if nil != err {
					log.Error(err)
				}
				l.requestCh <- Request{
					Conn: conn,
					Err:  err,
				}
			}
		}
	}()

	return err
}

// Addr returns the network address.
func (l *Listener) Addr() net.Addr {
	if nil == l.conn {
		return nil
	}
	return l.conn.Addr()
}

// Status returns whether the network connection has been closed.
func (l *Listener) Status() ListenerStatus {
	return l.status
}

// Protocol returns the protocol for this connection.
func (l *Listener) Protocol() string {
	return l.protocol
}

// Port returns the target port for this connection..
func (l *Listener) Port() int32 {
	return l.port
}

// String returns the target protocol type and address of the dialed service.
func (l *Listener) String() string {
	return fmt.Sprintf(
		"%s %d",
		l.Protocol(),
		l.Port(),
	)
}
