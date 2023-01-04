package metrics

import (
	"fmt"

	"github.com/hikhvar/mqtt2prometheus/pkg/config"
	gojsonq "github.com/thedevsaddam/gojsonq/v2"
)

type Extractor func(topic string, payload []byte, deviceID string) (MetricCollection, error)

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

			// Find a valid metrics config
			config, found := p.findMetricConfig(path, deviceID)
			if !found {
				continue
			}

			m, err := p.parseMetric(config, rawValue)
			if err != nil {
				return nil, fmt.Errorf("failed to parse valid metric value: %w", err)
			}
			m.Topic = topic
			mc = append(mc, m)
		}
		return mc, nil
	}
}

func NewMetricPerTopicExtractor(p Parser, metricNameRegex *config.Regexp) Extractor {
	return func(topic string, payload []byte, deviceID string) (MetricCollection, error) {
		metricName := metricNameRegex.GroupValue(topic, config.MetricNameRegexGroup)
		if metricName == "" {
			return nil, fmt.Errorf("failed to find valid metric in topic path")
		}

		// Find a valid metrics config
		config, found := p.findMetricConfig(metricName, deviceID)
		if !found {
			return nil, nil
		}

		m, err := p.parseMetric(config, string(payload))
		if err != nil {
			return nil, fmt.Errorf("failed to parse metric: %w", err)
		}
		m.Topic = topic
		return MetricCollection{m}, nil
	}
}
