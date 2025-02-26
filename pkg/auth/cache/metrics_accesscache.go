package cache

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	StatusQueuedLabel  string = "queued"
	StatusSkippedLabel string = "skipped"
)

var (
	_ AccessCacheMetrics = &accessCacheMetrics{}
	_ AccessCacheMetrics = &NoOpAccessCacheMetrics{}
)

// AccessCacheMetrics exposes functions to collect AccessCache's metrics
type AccessCacheMetrics interface {
	prometheus.Collector

	// CollectRequestMetrics collects metrics on synchronization requests
	CollectRequestMetrics(Event, bool)
	// CollectSynchMetrics collects metrics on synchronization runs
	CollectSynchMetrics(AccessData, error)
}

// NoOpAccessCacheMetrics is used to disable AccessCache's metrics
type NoOpAccessCacheMetrics struct{}

func (m *NoOpAccessCacheMetrics) Collect(_ chan<- prometheus.Metric)        {}
func (m *NoOpAccessCacheMetrics) Describe(_ chan<- *prometheus.Desc)        {}
func (m *NoOpAccessCacheMetrics) CollectRequestMetrics(_ Event, _ bool)     { return }
func (m *NoOpAccessCacheMetrics) CollectSynchMetrics(_ AccessData, _ error) { return }

// accessCacheMetrics is used to collect AccessCache's metrics
type accessCacheMetrics struct {
	// subjectCounter counts the subjects in the cache
	subjectCounter prometheus.Gauge
	// subjectNamespacePairsCounter counts the (subject, namespace) pairs in the cache
	subjectNamespacePairsCounter prometheus.Gauge
	// synchGauge counts the number of cache synchronization
	synchGauge *prometheus.CounterVec

	// resourceRequestsGauge counts the number of cache synchronization
	// requested as a consequence of resource events
	resourceRequestsGauge *prometheus.CounterVec
	// timeRequestsGauge counts the number of cache synchronization
	// that has been requested as resync period elapsed
	timeRequestsGauge *prometheus.CounterVec
}

// NewAccessCacheMetrics builds a new accessCacheMetrics
func NewAccessCacheMetrics() AccessCacheMetrics {
	return &accessCacheMetrics{
		subjectCounter: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "namespace_lister",
			Subsystem: "accesscache",
			Name:      "subjects",
			Help:      "Subjects in the cache",
		}),
		subjectNamespacePairsCounter: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "namespace_lister",
			Subsystem: "accesscache",
			Name:      "subject_namespace_pairs",
			Help:      "(Subject, Namespace) pairs in the cache",
		}),
		synchGauge: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "namespace_lister",
			Subsystem: "accesscache",
			Name:      "synch_op_total",
			Help:      "synchronization operations",
		}, []string{
			"status",
			"error",
		}),
		timeRequestsGauge: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "namespace_lister",
			Subsystem: "accesscache",
			Name:      "time_requests_total",
			Help:      "synchronization requests triggered when resync period elapses",
		}, []string{
			"status",
		}),
		resourceRequestsGauge: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "namespace_lister",
			Subsystem: "accesscache",
			Name:      "resource_requests_total",
			Help:      "synchronization requests triggered by events on watched resources",
		}, []string{
			"status",
		}),
	}
}

func (m *accessCacheMetrics) Collect(ch chan<- prometheus.Metric) {
	m.resourceRequestsGauge.Collect(ch)
	m.timeRequestsGauge.Collect(ch)

	m.subjectCounter.Collect(ch)
	m.subjectNamespacePairsCounter.Collect(ch)
	m.synchGauge.Collect(ch)
}

func (m *accessCacheMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.resourceRequestsGauge.Describe(ch)
	m.timeRequestsGauge.Describe(ch)

	m.subjectCounter.Describe(ch)
	m.subjectNamespacePairsCounter.Describe(ch)
	m.synchGauge.Describe(ch)
}

func (m *accessCacheMetrics) CollectRequestMetrics(event Event, queued bool) {
	// increment the appropriate requests gauge
	status := m.getStatusLabel(queued)

	switch event {
	case timeTriggeredEvent:
		m.collectTimeTriggeredRequestMetrics(status)
	default:
		m.collectResourceEventRequestMetrics(status)
	}
}

func (m *accessCacheMetrics) getStatusLabel(queued bool) string {
	if queued {
		return StatusQueuedLabel
	}
	return StatusSkippedLabel
}

func (m *accessCacheMetrics) collectTimeTriggeredRequestMetrics(status string) {
	m.timeRequestsGauge.With(prometheus.Labels{"status": status}).Inc()
}

func (m *accessCacheMetrics) collectResourceEventRequestMetrics(status string) {
	// set labels
	labels := prometheus.Labels{"status": status}

	// increment the number of requests triggered by events on resources
	m.resourceRequestsGauge.With(labels).Inc()
}

func (s *accessCacheMetrics) CollectSynchMetrics(cacheData AccessData, err error) {
	if err != nil {
		// increment failed synchronization counter
		s.synchGauge.With(prometheus.Labels{"status": "failed", "error": err.Error()}).Inc()
		return
	}

	// increment successful synchronizations counter
	s.synchGauge.With(prometheus.Labels{"status": "completed", "error": ""}).Inc()

	// update subjects in cache
	s.subjectCounter.Set(float64(len(cacheData)))

	// update number of (subject, namespace) pairs
	subNsPairCount := 0
	for _, v := range cacheData {
		subNsPairCount += len(v)
	}
	s.subjectNamespacePairsCounter.Set(float64(subNsPairCount))
}
