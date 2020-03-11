package metrics

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/hikhvar/mqtt2prometheus/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
)

type Ingest struct {
	validMetrics  map[string]config.MetricConfig
	collector     Collector
	MessageMetric *prometheus.CounterVec
}

var validNumber = regexp.MustCompile(`^[0-9.]+$`)

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

type MQTTPayload map[string]interface{}

func (i *Ingest) store(deviceID string, rawMetrics MQTTPayload) error {
	var mc MetricCollection

	for metricName, value := range rawMetrics {
		if cfg, found := i.validMetrics[metricName]; found {
			var floatValue float64
			var isFloat bool
			var err error
			floatValue, isFloat = value.(float64)
			if !isFloat {
				stringValue, isString := value.(string)

				if !isString || ! validNumber.MatchString(stringValue) {
					return fmt.Errorf("got data with unexpectd type: %T ('%s')", value, value)
				}

				floatValue, err = strconv.ParseFloat(stringValue, 64)
				if err != nil {
					return fmt.Errorf("got data with unexpectd type: %T ('%s') and failed to parse to float", value, value)
				}
			}

			mc = append(mc, Metric{
				Description: cfg.PrometheusDescription(),
				Value:       floatValue,
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
			return
		}
		err = i.store(deviceId, rawMetrics)
		if err != nil {
			errChan <- fmt.Errorf("could not store metrics '%s' on topic %s: %s", string(m.Payload()), m.Topic(), err.Error())
			i.MessageMetric.WithLabelValues("storeError", m.Topic()).Inc()
			return
		}
		i.MessageMetric.WithLabelValues("success", m.Topic()).Inc()
	}

}
