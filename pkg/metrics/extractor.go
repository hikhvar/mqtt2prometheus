package metrics

import (
	"fmt"
	"regexp"

	"github.com/hikhvar/mqtt2prometheus/pkg/config"
	gojsonq "github.com/thedevsaddam/gojsonq/v2"
)

type Extractor func(topic string, payload []byte, deviceID string) (MetricCollection, error)

// metricID returns a deterministic identifier per metic config which is safe to use in a file path.
func metricID(topic, metric, deviceID, promName string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9]`)
	deviceID = re.ReplaceAllString(deviceID, "_")
	topic = re.ReplaceAllString(topic, "_")
	metric = re.ReplaceAllString(metric, "_")
	promName = re.ReplaceAllString(promName, "_")
	return fmt.Sprintf("%s-%s-%s-%s", deviceID, topic, metric, promName)
}

func NewJSONObjectExtractor(p Parser) Extractor {
	return func(topic string, payload []byte, deviceID string) (MetricCollection, error) {
		var mc MetricCollection
		parsed := gojsonq.New(gojsonq.SetSeparator(p.separator)).FromString(string(payload))

		for path := range p.config() {
			rawValue := parsed.Find(path)
			parsed.Reset()
			if rawValue == nil {
				continue
			}

			// Find all valid metric configs
			for _, config := range p.findMetricConfigs(path, deviceID) {
				id := metricID(topic, path, deviceID, config.PrometheusName)
				m, err := p.parseMetric(config, id, rawValue)
				if err != nil {
					return nil, fmt.Errorf("failed to parse valid value from '%v' for metric %q: %w", rawValue, config.PrometheusName, err)
				}
				m.Topic = topic
				mc = append(mc, m)
			}
		}
		return mc, nil
	}
}

func NewMetricPerTopicExtractor(p Parser, metricNameRegex *config.Regexp) Extractor {
	return func(topic string, payload []byte, deviceID string) (MetricCollection, error) {
		var mc MetricCollection
		metricName := metricNameRegex.GroupValue(topic, config.MetricNameRegexGroup)
		if metricName == "" {
			return nil, fmt.Errorf("failed to find valid metric in topic path")
		}

		// Find all valid metric configs
		for _, config := range p.findMetricConfigs(metricName, deviceID) {
			var rawValue interface{}
			if config.PayloadField != "" {
				parsed := gojsonq.New(gojsonq.SetSeparator(p.separator)).FromString(string(payload))
				rawValue = parsed.Find(config.PayloadField)
				parsed.Reset()
				if rawValue == nil {
					return nil, fmt.Errorf("failed to extract field %q from payload %q for metric %q", config.PayloadField, payload, metricName)
				}
			} else {
				rawValue = string(payload)
			}

			id := metricID(topic, metricName, deviceID, config.PrometheusName)
			m, err := p.parseMetric(config, id, rawValue)
			if err != nil {
				return nil, fmt.Errorf("failed to parse valid value from '%v' for metric %q: %w", rawValue, config.PrometheusName, err)
			}
			m.Topic = topic
			mc = append(mc, m)
		}
		return mc, nil
	}
}
