package metrics

import (
	"errors"

	"path/filepath"

	"encoding/json"

	"fmt"

	"log"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/hikhvar/mqtt2prometheus/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
)

var NoValidPayload = errors.New("no valid MQTT payload")

type Ingest struct {
	validMetrics  map[string]config.MetricConfig
	collector     Collector
	MessageMetric *prometheus.CounterVec
}

func NewIngest(collector Collector, metrics []config.MetricConfig) *Ingest {
	valid := make(map[string]config.MetricConfig)
	for i := range metrics {
		key := metrics[i].MQTTName
		valid[key] = metrics[i]
	}
	return &Ingest{
		validMetrics: valid,
		collector:    collector,
		MessageMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "received_messages",
				Help: "received messages per topic and status",
			}, []string{"status", "topic"},
		),
	}
}

type MQTTPayload map[string]float64

func (i *Ingest) store(deviceID string, rawMetrics MQTTPayload) error {
	var mc MetricCollection
	for metricName, value := range rawMetrics {
		if cfg, found := i.validMetrics[metricName]; found {
			mc = append(mc, Metric{
				Description: cfg.PrometheusDescription(),
				Value:       value,
				ValueType:   cfg.PrometheusValueType(),
			})
		}
	}
	i.collector.Observe(deviceID, mc)
	return nil
}

func (i *Ingest) SetupSubscriptionHandler(errChan chan<- error) mqtt.MessageHandler {
	return func(c mqtt.Client, m mqtt.Message) {
		log.Printf("Got message '%s' on topic %s\n", string(m.Payload()), m.Topic())
		deviceId := filepath.Base(m.Topic())
		var rawMetrics MQTTPayload
		err := json.Unmarshal(m.Payload(), &rawMetrics)
		if err != nil {
			errChan <- fmt.Errorf("could not decode message '%s' on topic %s: %s", string(m.Payload()), m.Topic(), err.Error())
			i.MessageMetric.WithLabelValues("decodeError", m.Topic()).Desc()
		}
		err = i.store(deviceId, rawMetrics)
		if err != nil {
			errChan <- fmt.Errorf("could not store metrics '%s' on topic %s: %s", string(m.Payload()), m.Topic(), err.Error())
			i.MessageMetric.WithLabelValues("storeError", m.Topic()).Inc()
		}
		i.MessageMetric.WithLabelValues("success", m.Topic()).Inc()
	}

}
