package metricsutil

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

func GetMetricFamilyFromCollector(collector prometheus.Collector, name string) (*dto.MetricFamily, error) {
	reg := prometheus.NewRegistry()
	if err := reg.Register(collector); err != nil {
		return nil, err
	}

	mff, err := reg.Gather()
	if err != nil {
		return nil, err
	}

	for _, mf := range mff {
		if mf.GetName() == name {
			return mf, nil
		}
	}
	return nil, errors.New("metric family not found")
}

func GetVector(collector prometheus.Collector, name string) (model.Vector, error) {
	mf, err := GetMetricFamilyFromCollector(collector, name)
	if err != nil {
		return nil, err
	}

	return expfmt.ExtractSamples(&expfmt.DecodeOptions{Timestamp: model.Now()}, mf)
}

// GetHistogramSum returns the sum of all observed values for a histogram metric with matching labels.
// For histograms, this extracts the _sum metric.
func GetHistogramSum(collector prometheus.Collector, name string, labelMatchers map[string]string) (float64, error) {
	vec, err := GetVector(collector, name)
	if err != nil {
		return 0, err
	}

	sumName := name + "_sum"
	for _, sample := range vec {
		sn := string(sample.Metric[model.MetricNameLabel])
		if sn == sumName && matchesLabels(sample.Metric, labelMatchers) {
			return float64(sample.Value), nil
		}
	}

	return 0, errors.New("histogram sum not found with matching labels")
}

// GetHistogramCount returns the count of observations for a histogram metric with matching labels.
// For histograms, this extracts the _count metric.
func GetHistogramCount(collector prometheus.Collector, name string, labelMatchers map[string]string) (float64, error) {
	vec, err := GetVector(collector, name)
	if err != nil {
		return 0, err
	}

	countName := name + "_count"
	for _, sample := range vec {
		sn := string(sample.Metric[model.MetricNameLabel])
		if sn == countName && matchesLabels(sample.Metric, labelMatchers) {
			return float64(sample.Value), nil
		}
	}

	return 0, errors.New("histogram count not found with matching labels")
}

// matchesLabels checks if a metric's labels match all the provided label matchers.
func matchesLabels(metric model.Metric, matchers map[string]string) bool {
	for key, value := range matchers {
		if metric[model.LabelName(key)] != model.LabelValue(value) {
			return false
		}
	}
	return true
}
