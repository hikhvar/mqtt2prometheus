// +build gofuzz

package metric_per_topic

import (
	"fmt"
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
			ErrorValue: floatP(12333),
			StringValueMapping: &config.StringValueMappingConfig{
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
	json := metrics.NewMetricPerTopicExtractor(p, config.MustNewRegexp("shellies/(?P<deviceid>.*)/sensor/(?P<metricname>.*)"))

	name := "enabled"
	consumed := 0
	if len(data) > 0 {

		name = []string{"temperature", "enabled", "kartoffel"}[data[0]%3]
		consumed += 1

	}
	mc, err := json(fmt.Sprintf("shellies/bar/sensor/%s", name), data[consumed:], "bar")
	if err != nil && len(mc) > 0 {
		return 1
	}
	return 0
}

func floatP(f float64) *float64 {
	return &f
}
