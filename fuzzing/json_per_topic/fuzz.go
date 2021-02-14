// +build gofuzz

package json

import (
	"github.com/hikhvar/mqtt2prometheus/pkg/config"
	"github.com/hikhvar/mqtt2prometheus/pkg/metrics"
)

func Fuzz(data []byte) int {
	p := metrics.NewParser([]config.MetricConfig{
		{
			PrometheusName: "temperature",
			ValueType:      "gauge",
		},
		{
			PrometheusName: "enabled",
			ValueType:      "gauge",
			StringValueMapping: &config.StringValueMappingConfig{
				ErrorValue: floatP(12333),
				Map: map[string]float64{
					"foo": 112,
					"bar": 2,
				},
			},
		},
		{
			PrometheusName: "kartoffeln",
			ValueType:      "counter",
		},
	})
	json := metrics.NewJSONObjectExtractor(p)
	mc, err := json("foo", data, "bar")
	if err != nil && len(mc) > 0 {
		return 1
	}
	return 0
}

func floatP(f float64) *float64 {
	return &f
}
