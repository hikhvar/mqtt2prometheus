package metrics

import (
	"fmt"
	"time"

	"github.com/hikhvar/mqtt2prometheus/pkg/config"
	gocache "github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type Collector interface {
	prometheus.Collector
	Observe(collection MetricCollection)
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
	DeviceID    string
}

type CacheItem struct {
	DeviceID string
	Metric   Metric
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

func (c *MemoryCachedCollector) Observe(collection MetricCollection) {
	for _, m := range collection {
		// TODO: Now that DeviceID is part of Metric, perhaps we don't need the CacheItem any more?
		item := CacheItem{
			DeviceID: m.DeviceID,
			Metric:   m,
		}
		c.cache.Set(fmt.Sprintf("%s-%s", m.DeviceID, m.Description.String()), item, gocache.DefaultExpiration)
	}
}

func (c *MemoryCachedCollector) Describe(ch chan<- *prometheus.Desc) {
	for i := range c.descriptions {
		ch <- c.descriptions[i]
	}
}

func (c *MemoryCachedCollector) Collect(mc chan<- prometheus.Metric) {
	for _, metricsRaw := range c.cache.Items() {
		item := metricsRaw.Object.(CacheItem)
		device, metric := item.DeviceID, item.Metric
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

		if metric.IngestTime.IsZero() {
			mc <- m
		} else {
			mc <- prometheus.NewMetricWithTimestamp(metric.IngestTime, m)
		}

	}
}
