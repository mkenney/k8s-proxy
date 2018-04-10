package k8s

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

/*
ChangeSet holds the beofre and after set of k8s services.
*/
type ChangeSet struct {
	Added   map[string]apiv1.Service
	Removed map[string]apiv1.Service
}

/*
Services maintains an up to date list of available kubernetes services.
*/
type Services struct {
	client    corev1.ServiceInterface
	interrupt chan bool

	svcMapMux sync.Mutex
	svcMap    chan (map[string]apiv1.Service)
}

/*
Map returns the current map of running services.
*/
func (services *Services) Map() map[string]apiv1.Service {
	return <-services.svcMap
}

/*
Stop ends the serviceWatcher goroutine.
*/
func (services *Services) Stop() {
	services.interrupt <- true && <-services.interrupt
}

/*
Watch starts the service watcher goroutine. frequency is the number of
seconds to wait between API update requests. Must be greater than 0.
Default value is 5.
*/
func (services *Services) Watch(frequency time.Duration) chan ChangeSet {
	if frequency <= 0 {
		frequency = 5
	}

	services.interrupt = make(chan bool)
	changeSetCh := make(chan ChangeSet)
	readyCh := make(chan bool)

	go func() {
		delay := frequency * time.Second // Poll the API and update every `delay` period
		last := time.Now()
		serviceMap := make(map[string]apiv1.Service)

		for {
			select {
			// Stop watching for changes.
			case <-services.interrupt:
				defer close(changeSetCh)
				defer close(services.interrupt)
				break

			default:
				// Block calls to retrieve the service map until it's
				// ready.
				if len(serviceMap) > 0 {
					select {
					case services.svcMap <- serviceMap:
					case <-time.After(delay - time.Now().Sub(last)):
					}
				}

				// Update the service data if the map is empty or once
				// per `frequency` period.
				if 0 == len(serviceMap) || time.Now().Sub(last) > delay {
					last = time.Now()

					// Fetch the service list from the k8s API
					svcs, err := services.client.List(metav1.ListOptions{})
					if nil != err {
						log.Error(err)
						continue
					}

					// Convert to a named map of services and compute
					// the differences between the current and previous
					// states
					svcMap := map[string]apiv1.Service{}
					for _, service := range svcs.Items {
						svcMap[service.Name] = service
					}
					changeSet := diffServices(serviceMap, svcMap)
					serviceMap = svcMap

					//log.Debugf(
					//	"updated available services; %d added, %d removed, %d total",
					//	len(changeSet.Added),
					//	len(changeSet.Removed),
					//	len(serviceMap),
					//)

					// Allow the launching routine to continue once the
					// initial data set has been loaded.
					if nil != readyCh {
						readyCh <- true
						readyCh = nil
					}

					// Signal that the services available in the cluster
					// have changed. Don't block longer than the
					// scheduled delay.
					select {
					case changeSetCh <- changeSet:
					case <-time.After(delay - time.Now().Sub(last)):
					}
				}
			}
		}
	}()

	// Block until the initial data set has been loaded.
	<-readyCh

	return changeSetCh
}

/*
diffServices returns the deltas between cur and new as a ChangeSet.
*/
func diffServices(cur, new map[string]apiv1.Service) ChangeSet {
	changes := ChangeSet{
		Added:   map[string]apiv1.Service{},
		Removed: map[string]apiv1.Service{},
	}

	for k, v := range cur {
		if _, ok := new[v.Name]; !ok {
			changes.Removed[k] = v
		}
	}
	for k, v := range new {
		if _, ok := cur[v.Name]; !ok {
			changes.Added[k] = v
		}
	}

	return changes
}
