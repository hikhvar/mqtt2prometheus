package metrics

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/hikhvar/mqtt2prometheus/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
)

type Ingest struct {
	metricConfigs map[string][]config.MetricConfig
	deviceIDRegex *config.Regexp
	collector     Collector
	MessageMetric *prometheus.CounterVec
}

func NewIngest(collector Collector, metrics []config.MetricConfig, deviceIDRegex *config.Regexp) *Ingest {
	cfgs := make(map[string][]config.MetricConfig)
	for i := range metrics {
		key := metrics[i].MQTTName
		cfgs[key] = append(cfgs[key], metrics[i])
	}
	return &Ingest{
		metricConfigs: cfgs,
		deviceIDRegex: deviceIDRegex,
		collector:     collector,
		MessageMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "received_messages",
				Help: "received messages per topic and status",
			}, []string{"status", "topic"},
		),
	}
}

// validMetric returns config matching the metric and deviceID
// Second return value indicates if config was found.
func (i *Ingest) validMetric(metric string, deviceID string) (config.MetricConfig, bool) {
	for _, c := range i.metricConfigs[metric] {
		if c.SensorNameFilter.Match(deviceID) {
			return c, true
		}
	}
	return config.MetricConfig{}, false
}

type MQTTPayload map[string]interface{}

func (i *Ingest) store(deviceID string, rawMetrics MQTTPayload) error {
	var mc MetricCollection

	for metricName, value := range rawMetrics {
		cfg, cfgFound := i.validMetric(metricName, deviceID)
		if !cfgFound {
			continue
		}

		var metricValue float64

		if boolValue, ok := value.(bool); ok {
			if boolValue {
				metricValue = 1
			} else {
				metricValue = 0
			}
		} else if strValue, ok := value.(string); ok {

			// If string value mapping is defined, use that
			if cfg.StringValueMapping != nil {

				floatValue, ok := cfg.StringValueMapping.Map[strValue]
				if ok {
					metricValue = floatValue
				} else if cfg.StringValueMapping.ErrorValue != nil {
					metricValue = *cfg.StringValueMapping.ErrorValue
				} else {
					return fmt.Errorf("got unexpected string data '%s'", strValue)
				}

			} else {

				// otherwise try to parse float
				floatValue, err := strconv.ParseFloat(strValue, 64)
				if err != nil {
					return fmt.Errorf("got data with unexpectd type: %T ('%s') and failed to parse to float", value, value)
				}
				metricValue = floatValue

			}

		} else if floatValue, ok := value.(float64); ok {
			metricValue = floatValue
		} else {
			return fmt.Errorf("got data with unexpectd type: %T ('%s')", value, value)
		}

		mc = append(mc, Metric{
			Description: cfg.PrometheusDescription(),
			Value:       metricValue,
			ValueType:   cfg.PrometheusValueType(),
			IngestTime:  time.Now(),
		})
	}
	i.collector.Observe(deviceID, mc)
	return nil
}

func (i *Ingest) SetupSubscriptionHandler(errChan chan<- error) mqtt.MessageHandler {
	return func(c mqtt.Client, m mqtt.Message) {
		log.Printf("Got message '%s' on topic %s\n", string(m.Payload()), m.Topic())
		deviceId := i.deviceID(m.Topic())
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

// deviceID uses the configured DeviceIDRegex to extract the device ID from the given mqtt topic path.
func (i *Ingest) deviceID(topic string) string {
	return i.deviceIDRegex.GroupValue(topic, config.DeviceIDRegexGroup)
}
