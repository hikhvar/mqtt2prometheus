package metrics

import (
	"go.uber.org/zap"
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
	logger       *zap.Logger
}

type Metric struct {
	Description *prometheus.Desc
	Value       float64
	ValueType   prometheus.ValueType
	IngestTime  time.Time
	Topic       string
}

type MetricCollection []Metric

func NewCollector(defaultTimeout time.Duration, possibleMetrics []config.MetricConfig, logger *zap.Logger) Collector {
	var descs []*prometheus.Desc
	for _, m := range possibleMetrics {
		descs = append(descs, m.PrometheusDescription())
	}
	return &MemoryCachedCollector{
		cache:        gocache.New(defaultTimeout, defaultTimeout*10),
		descriptions: descs,
		logger:       logger,
	}
}

func (c *MemoryCachedCollector) Observe(deviceID string, collection MetricCollection) {
	if len(collection) < 1 {
		return
	}
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
			if metric.Description == nil {
				c.logger.Warn("empty description", zap.String("topic", metric.Topic), zap.Float64("value", metric.Value))
			}
			m := prometheus.MustNewConstMetric(
				metric.Description,
				metric.ValueType,
				metric.Value,
				device,
				metric.Topic,
			)
			mc <- prometheus.NewMetricWithTimestamp(metric.IngestTime, m)
		}
	}
}
