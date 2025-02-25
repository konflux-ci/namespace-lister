package cache

import toolscache "k8s.io/client-go/tools/cache"

// EventType the event that's triggering a Request resynch
type EventType string

const (
	// ResourceAddedEventType represents a Creation of the event's Object
	ResourceAddedEventType EventType = "Add"
	// ResourceUpdatedEventType represents a Update on the event's Object
	ResourceUpdatedEventType EventType = "Update"
	// ResourceDeletedEventType represents a Deletion of the event's Object
	ResourceDeletedEventType EventType = "Delete"
)

// Event represents the event that's triggering the cache Resync.
// Mainly required for metrics gathering.
type Event struct {
	Object interface{}
	Type   EventType

	timeTriggered bool
}

// timeTriggeredEvent is the event used to request time-based synchronization
var timeTriggeredEvent = Event{timeTriggered: true}

// EventHandlerFuncs returns an EventHandlerFuncs to integrate with Informers.
func (s *SynchronizedAccessCache) EventHandlerFuncs() toolscache.ResourceEventHandlerFuncs {
	return toolscache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			s.Request(Event{Object: obj, Type: ResourceAddedEventType})
		},
		UpdateFunc: func(_, newObj interface{}) {
			s.Request(Event{Object: newObj, Type: ResourceUpdatedEventType})
		},
		DeleteFunc: func(obj interface{}) {
			s.Request(Event{Object: obj, Type: ResourceDeletedEventType})
		},
	}
}
