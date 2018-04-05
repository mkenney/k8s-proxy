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
	svcMap    map[string]apiv1.Service
}

/*
Map returns the current map of running services.
*/
func (services *Services) Map() map[string]apiv1.Service {
	return services.svcMap
}

/*
Stop ends the serviceWatcher goroutine.
*/
func (services *Services) Stop() {
	services.interrupt <- true
	<-services.interrupt
}

/*
Watch starts the serviceWatcher goroutine. updateFrequency is the number
of seconds to wait between API update requests. Must be greater than 0.
Default value is 5.
*/
func (services *Services) Watch(updateFrequency int) chan ChangeSet {
	if updateFrequency <= 0 {
		updateFrequency = 5
	}

	services.interrupt = make(chan bool)
	changeSetCh := make(chan ChangeSet)
	readyCh := make(chan bool)

	go func() {
		delay := time.Duration(updateFrequency) * time.Second // Poll the API and update every `delay` period
		last := time.Now()
		for {
			select {
			// Stop watching for changes.
			case <-services.interrupt:
				defer close(changeSetCh)
				defer close(services.interrupt)
				break

			// Update the service data when starting or once per delay
			// period.
			default:
				if 0 == len(services.svcMap) || time.Now().Sub(last) > delay {
					last = time.Now()

					// Fetch the service list from the k8s API
					svcs, err := services.client.List(metav1.ListOptions{})
					if nil != err {
						log.Warn(err.Error())
						continue
					}

					// Convert to a named map of services and compute
					// the differences between the current and previous
					// states
					svcMap := map[string]apiv1.Service{}
					for _, service := range svcs.Items {
						svcMap[service.Name] = service
					}
					changeSet := diffServices(services.svcMap, svcMap)

					// Update the current state
					services.svcMapMux.Lock()
					services.svcMap = svcMap
					services.svcMapMux.Unlock()
					log.Infof("updated available services; %d added, %d removed", len(changeSet.Added), len(changeSet.Removed))

					// Unblock the launching routine once the initial
					// data set has been loaded.
					if nil != readyCh {
						readyCh <- true
						readyCh = nil
					}

					// Signal that the services available in the cluster
					// have changed. Don't block longer than the
					// scheduled delay.
					if len(changeSet.Added) > 0 || len(changeSet.Removed) > 0 {
						select {
						case <-time.After(delay - time.Now().Sub(last)):
						case changeSetCh <- changeSet:
						}
					}
				}
				time.Sleep(10 * time.Millisecond)
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
