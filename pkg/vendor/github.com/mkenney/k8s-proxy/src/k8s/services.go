package k8s

import (
	"sync"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

type K8S struct {
	Client   corev1.CoreV1Interface
	Services *Services
}

/*
Services maintains an up to date list of available kubernetes services.
*/
type Services struct {
	client    corev1.ServiceInterface
	svcMapMux sync.Mutex
	svcMap    map[string]apiv1.Service
	watchCtl  chan *watchCtl
}
type watchCtl struct {
	stop bool
}

/*
New is the constructor for the Services struct.
*/
func New() (*K8S, error) {
	// create the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	k8s := &K8S{
		Client: client.CoreV1(),
	}
	k8s.Services = &Services{
		client: k8s.Client.Services(""),
	}

	return k8s, nil
}

/*
Map returns a map of running services.
*/
func (services *Services) Map() map[string]apiv1.Service {
	return services.svcMap
}

/*
Stop ends the serviceWatcher goroutine.
*/
func (services *Services) Stop() {
	services.watchCtl <- &watchCtl{stop: true}
	<-services.watchCtl
}

/*
Watch starts the serviceWatcher goroutine.
*/
func (services *Services) Watch() chan ChangeSet {
	services.watchCtl = make(chan *watchCtl)
	changeSetCh := make(chan ChangeSet, 1)

	go func() {
		var last time.Time

		changeSet := ChangeSet{}
		delay := 5 * time.Second

		for {
			select {

			// Stop watching for changes.
			case msg := <-services.watchCtl:
				if msg.stop {
					defer close(changeSetCh)
					defer close(services.watchCtl)
					break
				}

			// Update the service data when starting or once per delay
			// period.
			default:
				if 0 == len(services.svcMap) || time.Now().Sub(last) > delay {
					last = time.Now()

					svcs, err := services.client.List(metav1.ListOptions{})
					if nil != err {
						continue
					}

					svcMap := map[string]apiv1.Service{}
					for _, service := range svcs.Items {
						svcMap[service.Name] = service
					}

					changeSet = diff(services.svcMap, svcMap)
					services.svcMapMux.Lock()
					services.svcMap = svcMap
					services.svcMapMux.Unlock()

					// Signal that the services available in the cluster
					// have changed.
					select {
					case changeSetCh <- changeSet:
					}
				}
			}
		}
	}()

	return changeSetCh
}

/*
ChangeSet holds the beofre and after set of k8s services.
*/
type ChangeSet struct {
	Added   map[string]apiv1.Service
	Removed map[string]apiv1.Service
}

/*
diff returns the deltas between cur and new as a ChangeSet.
*/
func diff(cur, new map[string]apiv1.Service) ChangeSet {
	changes := ChangeSet{
		Added:   map[string]apiv1.Service{},
		Removed: map[string]apiv1.Service{},
	}
	mc := map[string]bool{}
	mn := map[string]bool{}

	for _, v := range cur {
		mc[v.Name] = true
	}
	for _, v := range new {
		mn[v.Name] = true
	}

	for k, v := range cur {
		if _, ok := mn[v.Name]; !ok {
			changes.Removed[k] = v
		}
	}
	for k, v := range new {
		if _, ok := mc[v.Name]; !ok {
			changes.Added[k] = v
		}
	}

	return changes
}
