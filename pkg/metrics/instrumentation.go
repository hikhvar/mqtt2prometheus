package metrics

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	storeError = "storeError"
	success    = "success"
)

var defaultInstrumentation = instrumentation{
	messageMetric: prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "received_messages",
			Help: "received messages per topic and status",
		}, []string{"status", "topic"},
	),
	connectedMetric: prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "mqtt2prometheus_connected",
			Help: "is the mqtt2prometheus exporter connected to the broker",
		},
	),
}

type instrumentation struct {
	messageMetric   *prometheus.CounterVec
	connectedMetric prometheus.Gauge
}

func (i *instrumentation) Collector() prometheus.Collector {
	return i
}

func (i *instrumentation) Describe(desc chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(i, desc)
}

func (i *instrumentation) Collect(metrics chan<- prometheus.Metric) {
	i.connectedMetric.Collect(metrics)
	i.messageMetric.Collect(metrics)
}

func (i *instrumentation) CountSuccess(topic string) {
	i.messageMetric.WithLabelValues(success, topic).Inc()
}

func (i *instrumentation) CountStoreError(topic string) {
	i.messageMetric.WithLabelValues(storeError, topic).Inc()
}

func (i *instrumentation) ConnectionLostHandler(client mqtt.Client, err error) {
	i.connectedMetric.Set(0)
}

func (i *instrumentation) OnConnectHandler(client mqtt.Client) {
	i.connectedMetric.Set(1)
}
