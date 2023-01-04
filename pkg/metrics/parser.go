package metrics

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hikhvar/mqtt2prometheus/pkg/config"
)

type Parser struct {
	separator string
	// Maps the mqtt metric name to a list of configs
	// The first that matches SensorNameFilter will be used
	metricConfigs map[string][]config.MetricConfig
}

var now = time.Now

func NewParser(metrics []config.MetricConfig, separator string) Parser {
	cfgs := make(map[string][]config.MetricConfig)
	for i := range metrics {
		key := metrics[i].MQTTName
		cfgs[key] = append(cfgs[key], metrics[i])
	}
	return Parser{
		separator:     separator,
		metricConfigs: cfgs,
	}
}

// Config returns the underlying metrics config
func (p *Parser) config() map[string][]config.MetricConfig {
	return p.metricConfigs
}

// validMetric returns config matching the metric and the deviceID extracted from the topic
// Third return value indicates if config was found.
func (p *Parser) findMetricConfig(metric string, topic string) (config.MetricConfig, string, bool) {
	for _, c := range p.metricConfigs[metric] {
		// use DeviceIDRegex to extract the device ID from the given mqtt topic path
		deviceId := c.DeviceIDRegex.GroupValue(topic, config.DeviceIDRegexGroup)
		if c.SensorNameFilter.Match(deviceId) {
			return c, deviceId, true
		}
	}
	return config.MetricConfig{}, "", false
}

// parseMetric parses the given value according to the given deviceID and metricPath. The config allows to
// parse a metric value according to the device ID.
func (p *Parser) parseMetric(cfg config.MetricConfig, value interface{}) (Metric, error) {
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

	if cfg.MQTTValueScale != 0 {
		metricValue = metricValue * cfg.MQTTValueScale
	}

	var ingestTime time.Time
	if !cfg.OmitTimestamp {
		ingestTime = now()
	}

	return Metric{
		Description: cfg.PrometheusDescription(),
		Value:       metricValue,
		ValueType:   cfg.PrometheusValueType(),
		IngestTime:  ingestTime,
	}, nil
}
