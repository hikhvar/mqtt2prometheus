package metrics

import (
	"fmt"
	"log"
	"strconv"
	"time"

	gojsonq "github.com/thedevsaddam/gojsonq/v2"

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

func (i *Ingest) store(deviceID string, payload []byte) error {
	var mc MetricCollection
	parsed := gojsonq.New().FromString(string(payload))

	for path := range i.metricConfigs {
		rawValue := parsed.Find(path)
		parsed.Reset()
		fmt.Printf("query path: %q data: %v\n", path, rawValue)

		m, err := i.parseMetric(path, deviceID, rawValue)
		if err != nil {
			return fmt.Errorf("failed to parse valid metric value: %w", err)
		}
		mc = append(mc, m)
	}

	i.collector.Observe(deviceID, mc)
	return nil
}

func (i *Ingest) parseMetric(metricPath string, deviceID string, value interface{}) (Metric, error) {
	cfg, cfgFound := i.validMetric(metricPath, deviceID)
	if !cfgFound {
		return Metric{}, nil
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
				return Metric{}, fmt.Errorf("got unexpected string data '%s'", strValue)
			}

		} else {

			// otherwise try to parse float
			floatValue, err := strconv.ParseFloat(strValue, 64)
			if err != nil {
				return Metric{}, fmt.Errorf("got data with unexpectd type: %T ('%s') and failed to parse to float", value, value)
			}
			metricValue = floatValue

		}

	} else if floatValue, ok := value.(float64); ok {
		metricValue = floatValue
	} else {
		return Metric{}, fmt.Errorf("got data with unexpectd type: %T ('%s')", value, value)
	}
	return Metric{
		Description: cfg.PrometheusDescription(),
		Value:       metricValue,
		ValueType:   cfg.PrometheusValueType(),
		IngestTime:  time.Now(),
	}, nil
}

func (i *Ingest) SetupSubscriptionHandler(errChan chan<- error) mqtt.MessageHandler {
	return func(c mqtt.Client, m mqtt.Message) {
		log.Printf("Got message '%s' on topic %s\n", string(m.Payload()), m.Topic())
		deviceId := i.deviceID(m.Topic())

		err := i.store(deviceId, m.Payload())
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
