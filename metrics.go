package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewDefaultRegistry() *prometheus.Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{
			Namespace: "namespace_lister",
		}),
	)

	return reg
}

type metrics struct {
	requestTiming  *prometheus.HistogramVec
	requestCounter *prometheus.CounterVec
	responseSize   *prometheus.HistogramVec
	inFlightGauge  prometheus.Gauge
}

func newMetrics(reg prometheus.Registerer) metrics {
	requestTiming := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "namespace_lister",
			Subsystem: "api",
			Name:      "latency",
			Help:      "Latency of requests",
			Buckets: []float64{
				0.000000001, // nanoseconds
				0.0000000025,
				0.000000005,
				0.00000001,
				0.000000025,
				0.00000005,
				0.0000001,
				0.00000025,
				0.0000005,
				0.000001, // microseconds
				0.0000025,
				0.000005,
				0.00001,
				0.000025,
				0.00005,
				0.0001,
				0.00025,
				0.0005,
				0.001, // milliseconds
				0.0025,
				0.005,
				0.01,
				0.025,
				0.05,
				0.1,
				0.25,
				0.5,
				1.0,
				2.0,
				5.0,
				10.0,
				20.0,
				30.0,
				60.0,
			},
		}, []string{"code", "method"})

	requestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "namespace_lister",
			Subsystem: "api",
			Name:      "counter",
			Help:      "Number of requests completed",
		}, []string{"code", "method"})

	responseSize := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "namespace_lister",
			Subsystem: "api",
			Name:      "response_size",
			Help:      "Size of responses",
			Buckets: []float64{
				1.0,
				2.0,
				5.0,
				10.0,
				20.0,
				50.0,
				100.0,
				200.0,
				500.0,
				1000.0,
				2000.0,
				5000.0,
				10000.0,
				20000.0,
				50000.0,
				100000.0,
				200000.0,
				500000.0,
			},
		}, []string{"code", "method"})

	inFlightGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "namespace_lister",
			Subsystem: "api",
			Name:      "requests_in_flight",
			Help:      "Number of requests currently processing",
		})

	reg.MustRegister(requestTiming, requestCounter, responseSize, inFlightGauge)

	return metrics{
		requestTiming:  requestTiming,
		requestCounter: requestCounter,
		responseSize:   responseSize,
		inFlightGauge:  inFlightGauge,
	}
}

func AddMetricsMiddleware(reg prometheus.Registerer, handler http.Handler) http.Handler {
	if reg == nil {
		return handler
	}

	m := newMetrics(reg)
	return promhttp.InstrumentHandlerDuration(
		m.requestTiming,
		promhttp.InstrumentHandlerCounter(
			m.requestCounter,
			promhttp.InstrumentHandlerResponseSize(
				m.responseSize,
				promhttp.InstrumentHandlerInFlight(
					m.inFlightGauge,
					handler))))
}
