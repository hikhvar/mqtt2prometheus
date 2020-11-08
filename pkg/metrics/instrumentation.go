package metrics

import "github.com/prometheus/client_golang/prometheus"

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
}

type instrumentation struct {
	messageMetric *prometheus.CounterVec
}

func (i *instrumentation) MessageMetric() *prometheus.CounterVec {
	return i.messageMetric
}

func (i *instrumentation) CountSuccess(topic string) {
	i.messageMetric.WithLabelValues(success, topic).Inc()
}

func (i *instrumentation) CountStoreError(topic string) {
	i.messageMetric.WithLabelValues(storeError, topic).Inc()
}
