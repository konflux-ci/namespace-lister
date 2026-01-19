package cache

import (
	"strings"

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
	CollectSynchMetrics(float64, AccessData, error)
}

// NoOpAccessCacheMetrics is used to disable AccessCache's metrics
type NoOpAccessCacheMetrics struct{}

func (m *NoOpAccessCacheMetrics) Collect(_ chan<- prometheus.Metric)                   {}
func (m *NoOpAccessCacheMetrics) Describe(_ chan<- *prometheus.Desc)                   {}
func (m *NoOpAccessCacheMetrics) CollectRequestMetrics(_ Event, _ bool)                {}
func (m *NoOpAccessCacheMetrics) CollectSynchMetrics(_ float64, _ AccessData, _ error) {}

// accessCacheMetrics is used to collect AccessCache's metrics
type accessCacheMetrics struct {
	// subjectCounter counts the subjects in the cache
	subjectCounter prometheus.Gauge
	// subjectNamespacePairsCounter counts the (subject, namespace) pairs in the cache
	subjectNamespacePairsCounter *prometheus.GaugeVec
	// synchGauge counts the number of cache synchronization
	synchGauge *prometheus.CounterVec
	//synchDuration tracks duration of each synch cycle
	synchDuration *prometheus.HistogramVec

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
		subjectNamespacePairsCounter: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "namespace_lister",
			Subsystem: "accesscache",
			Name:      "subject_namespace_pairs",
			Help:      "(Subject, Namespace) pairs in the cache",
		}, []string{"subject_gk"}),
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
		synchDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "namespace_lister",
			Subsystem: "accesscache",
			Name:      "synch_duration_milliseconds",
			Help:      "duration of the synchronization routine",
			Buckets: []float64{
				10,
				50,
				100,
				200,
				500,
				1000,           // 1s
				10 * 1000,      // 10s
				30 * 1000,      // 30s
				60 * 1000,      // 60s
				2 * 60 * 1000,  // 2m
				5 * 60 * 1000,  // 5m
				10 * 60 * 1000, // 10m
				30 * 60 * 1000, // 30m
				60 * 60 * 1000, // 60m
			},
		}, []string{
			"status",
		}),
	}
}

func (m *accessCacheMetrics) Collect(ch chan<- prometheus.Metric) {
	m.resourceRequestsGauge.Collect(ch)
	m.timeRequestsGauge.Collect(ch)
	m.synchDuration.Collect(ch)

	m.subjectCounter.Collect(ch)
	m.subjectNamespacePairsCounter.Collect(ch)
	m.synchGauge.Collect(ch)
}

func (m *accessCacheMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.resourceRequestsGauge.Describe(ch)
	m.timeRequestsGauge.Describe(ch)
	m.synchDuration.Describe(ch)

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

func (s *accessCacheMetrics) CollectSynchMetrics(duration float64, cacheData AccessData, err error) {
	if err != nil {
		// store synch duration
		s.synchDuration.With(prometheus.Labels{"status": "failed"}).Observe(duration)

		// increment failed synchronization counter
		s.synchGauge.With(prometheus.Labels{"status": "failed", "error": err.Error()}).Inc()
		return
	}

	// store synch duration
	s.synchDuration.With(prometheus.Labels{"status": "completed"}).Observe(duration)

	// increment successful synchronizations counter
	s.synchGauge.With(prometheus.Labels{"status": "completed", "error": ""}).Inc()

	// update subjects in cache
	s.subjectCounter.Set(float64(len(cacheData)))

	// update number of (subject, namespace) pairs
	gkNsPairCount := map[string]int{}
	for s, v := range cacheData {
		sgk := strings.TrimLeft(s.APIGroup+"/"+s.Kind, "/")
		gkNsPairCount[sgk] = gkNsPairCount[sgk] + len(v)
	}

	for gk, n := range gkNsPairCount {
		s.subjectNamespacePairsCounter.
			With(prometheus.Labels{"subject_gk": gk}).
			Set(float64(n))
	}
}
