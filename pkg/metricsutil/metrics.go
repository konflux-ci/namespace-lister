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
