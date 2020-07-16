package metrics

import (
	"time"

	"github.com/hikhvar/mqtt2prometheus/pkg/config"
	gocache "github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
)

const DefaultTimeout = 0

type Collector interface {
	prometheus.Collector
	Observe(deviceID string, collection MetricCollection)
}

type MemoryCachedCollector struct {
	cache        *gocache.Cache
	descriptions []*prometheus.Desc
}

type Metric struct {
	Description *prometheus.Desc
	Value       float64
	ValueType   prometheus.ValueType
	IngestTime  time.Time
}

type MetricCollection []Metric

func NewCollector(defaultTimeout time.Duration, possibleMetrics []config.MetricConfig) Collector {
	var descs []*prometheus.Desc
	for _, m := range possibleMetrics {
		descs = append(descs, m.PrometheusDescription())
	}
	return &MemoryCachedCollector{
		cache:        gocache.New(defaultTimeout, defaultTimeout*10),
		descriptions: descs,
	}
}

func (c *MemoryCachedCollector) Observe(deviceID string, collection MetricCollection) {
	c.cache.Set(deviceID, collection, DefaultTimeout)
}

func (c *MemoryCachedCollector) Describe(ch chan<- *prometheus.Desc) {
	for i := range c.descriptions {
		ch <- c.descriptions[i]
	}
}

func (c *MemoryCachedCollector) Collect(mc chan<- prometheus.Metric) {
	for device, metricsRaw := range c.cache.Items() {
		metrics := metricsRaw.Object.(MetricCollection)
		for _, metric := range metrics {
			m, err := prometheus.NewConstMetric(
				metric.Description,
				metric.ValueType,
				metric.Value,
				device,
			)
			if err != nil {
				panic(err)
			}
			mc <- prometheus.NewMetricWithTimestamp(metric.IngestTime, m)
		}
	}
}
