package proxy

import (
	"fmt"
	"net"

	"github.com/bdlm/log"
)

type ListenerStatus int

const (
	Ready ListenerStatus = iota
	Listening
	Closed
)

//var netConfig = &net.Config{InsecureSkipVerify: true}

// NewListener creates a new reverse proxy to forward traffic through.
func NewListener(network string, port int32, requestCh chan Request) *Listener {
	return &Listener{
		closeCh:   make(chan error),
		network:   network,
		port:      port,
		requestCh: requestCh,
		status:    Ready,
	}
}

// Listener defines the network listener for services using a particular
// network and port.
type Listener struct {
	closeCh   chan error
	conn      net.Listener
	network   string
	port      int32
	requestCh chan Request
	status    ListenerStatus
}

// Listen starts the network listener.
func (l *Listener) Listen() error {
	var err error

	if nil == l.conn || Closed == l.status {
		l.conn, err = net.Listen(l.Network(), fmt.Sprintf(":%d", l.Port()))
		if nil == err {
			l.status = Listening

			// Start the listen loop
			go func() {
				for {
					select {
					case <-l.closeCh:
						l.closeCh <- l.conn.Close()
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
		}
	}

	return err
}

// Addr returns the network address.
func (l *Listener) Addr() net.Addr {
	if nil == l.conn {
		return nil
	}
	return l.conn.Addr()
}

// Close stops the network listener and removes the port binding from the
// k8s service.
func (l *Listener) Close() error {
	// close the listener and end the listen loop.
	l.closeCh <- nil
	err := <-l.closeCh

	// do something?
	if nil != err {
		log.Error(err)
	}

	// modify the k8s-proxy service to remove the closed network+port...

	l.status = Closed
	return err
}

// Status returns whether the network connection has been closed.
func (l *Listener) Status() ListenerStatus {
	return l.status
}

// Network returns the network for this connection.
func (l *Listener) Network() string {
	return l.network
}

// Port returns the target port for this connection..
func (l *Listener) Port() int32 {
	return l.port
}

// String returns the target network type and address of the dialed service.
func (l *Listener) String() string {
	return fmt.Sprintf(
		"%s %d",
		l.Network(),
		l.Port(),
	)
}
