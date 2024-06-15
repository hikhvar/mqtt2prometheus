package metrics

import (
	"fmt"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/hikhvar/mqtt2prometheus/pkg/config"
	gocache "github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
)

type Collector interface {
	prometheus.Collector
	Observe(deviceID string, collection MetricCollection)
}

type MemoryCachedCollector struct {
	cache        *gocache.Cache
	descriptions []*prometheus.Desc
	logger       log.Logger
}

type Metric struct {
	Description *prometheus.Desc
	Value       float64
	ValueType   prometheus.ValueType
	IngestTime  time.Time
	Topic       string
}

type CacheItem struct {
	DeviceID string
	Metric   Metric
}

type MetricCollection []Metric

func NewCollector(defaultTimeout time.Duration, possibleMetrics []config.MetricConfig, logger log.Logger) Collector {
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
	for _, m := range collection {
		item := CacheItem{
			DeviceID: deviceID,
			Metric:   m,
		}
		c.cache.Set(fmt.Sprintf("%s-%s", deviceID, m.Description.String()), item, gocache.DefaultExpiration)
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
			level.Warn(c.logger).Log("msg", "empty description", "topic", metric.Topic, "value", metric.Value)
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
