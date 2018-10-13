package proxy

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	errs "github.com/bdlm/errors"
	"github.com/bdlm/log"
	"github.com/mkenney/k8s-proxy/internal/codes"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
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
func NewListener(
	ctx context.Context,
	protocol string,
	port int32,
	requestCh chan Request,
	svcs corev1.ServiceInterface,
) (*Listener, error) {
	var svcExists bool
	svcKey := fmt.Sprintf("%s:%d", protocol, port)

	svc, err := svcs.Get("k8s-proxy", metav1.GetOptions{})
	if nil != err {
		return nil, err
	}

	for _, svcPort := range svc.Spec.Ports {
		// Determine network protocol.
		lstnProtocol := "tcp"
		if "" != svcPort.Protocol {
			lstnProtocol = strings.ToLower(string(svcPort.Protocol))
		}

		// Determine port for receiving traffic.
		lstnPort := svcPort.Port
		if 0 > svcPort.TargetPort.IntVal {
			lstnPort = svcPort.TargetPort.IntVal
		}
		if 0 > svcPort.NodePort {
			lstnPort = svcPort.NodePort
		}

		// Check to see if this service port exists.
		if svcKey == fmt.Sprintf("%s:%d", lstnProtocol, lstnPort) {
			svcExists = true
			break
		}
	}

	// Modify the k8s-proxy service to add the protocol+port listener
	if !svcExists {
		svc.Spec.Ports = append(svc.Spec.Ports, apiv1.ServicePort{
			Name:     svcKey,
			Protocol: apiv1.Protocol(protocol),
			Port:     port,
			TargetPort: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: port,
			},
		})
		svcs.Update(svc)
	}

	id := listenerCounterNext()
	log.Infof("creating listener %d: %s:%d...", id, protocol, port)
	return &Listener{
		closeCh:   make(chan error),
		ctx:       ctx,
		id:        id,
		port:      port,
		protocol:  protocol,
		requestCh: requestCh,
		status:    Ready,
	}, nil
}

var listenerCounterMux sync.Mutex
var listenerCounter int

func listenerCounterNext() int {
	listenerCounterMux.Lock()
	listenerCounter++
	ret := listenerCounter
	listenerCounterMux.Unlock()
	return ret
}

// Listener defines the network listener for services using a particular
// protocol and port.
type Listener struct {
	closeCh   chan error
	conn      net.Listener
	ctx       context.Context
	id        int
	port      int32
	protocol  string
	requestCh chan Request
	status    ListenerStatus
}

// Listen starts the network listener.
func (l *Listener) Listen() error {
	var err error

	if Listening == l.status {
		return errs.New(codes.NetworkListenerListening, "network listner already running, skipping")
	}
	l.conn, err = net.Listen(l.Protocol(), fmt.Sprintf(":%d", l.Port()))
	if nil != err {
		return errs.Wrap(err, codes.NetworkListenerFailed, "could not open network listener")
	}
	l.status = Listening

	// Start the listen loop
	go func() {
		log.Info("starting the network listener...")
		ctx, cancel := context.WithCancel(l.ctx)
		for {
			select {
			case <-ctx.Done():
				log.Info("stopping the network listener...")
				log.Errorf("%-v", l.ctx.Err())
				cancel()
				return
			default:
				conn, err := l.conn.Accept()
				if nil != err {
					log.Errorf("%-v", err)
				}
				l.requestCh <- Request{
					Conn: conn,
					Err:  err,
				}
			}
		}
	}()

	return nil
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
