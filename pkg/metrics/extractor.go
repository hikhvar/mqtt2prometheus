package metrics

import (
	"errors"
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
			m, err := p.parseMetric(path, deviceID, rawValue)
			if err != nil {
				return nil, fmt.Errorf("failed to parse valid metric value: %w", err)
			}
			m.Topic = topic
			mc = append(mc, m)
		}
		return mc, nil
	}
}

func NewMetricPerTopicExtractor(p Parser, metricName *config.Regexp) Extractor {
	return func(topic string, payload []byte, deviceID string) (MetricCollection, error) {
		mName := metricName.GroupValue(topic, config.MetricNameRegexGroup)
		if mName == "" {
			return nil, fmt.Errorf("failed to find valid metric in topic path")
		}
		m, err := p.parseMetric(mName, deviceID, string(payload))
		if err != nil {
			if errors.Is(err, metricNotConfigured) {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to parse metric: %w", err)
		}
		m.Topic = topic
		return MetricCollection{m}, nil
	}
}
