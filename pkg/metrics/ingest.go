package metrics

import (
	"fmt"

	"go.uber.org/zap"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hikhvar/mqtt2prometheus/pkg/config"
)

type Ingest struct {
	instrumentation
	extractor     Extractor
	deviceIDRegex *config.Regexp
	collector     Collector
	logger        *zap.Logger
}

func NewIngest(collector Collector, extractor Extractor, deviceIDRegex *config.Regexp) *Ingest {

	return &Ingest{
		instrumentation: defaultInstrumentation,
		extractor:       extractor,
		deviceIDRegex:   deviceIDRegex,
		collector:       collector,
		logger:          config.ProcessContext.Logger(),
	}
}

func (i *Ingest) store(topic string, payload []byte) error {
	mc, err := i.extractor(topic, payload)
	if err != nil {
		return fmt.Errorf("failed to extract metric values from topic: %w", err)
	}
	i.collector.Observe(mc)
	return nil
}

func (i *Ingest) SetupSubscriptionHandler(errChan chan<- error) mqtt.MessageHandler {
	return func(c mqtt.Client, m mqtt.Message) {
		i.logger.Debug("Got message", zap.String("topic", m.Topic()), zap.String("payload", string(m.Payload())))
		err := i.store(m.Topic(), m.Payload())
		if err != nil {
			errChan <- fmt.Errorf("could not store metrics '%s' on topic %s: %s", string(m.Payload()), m.Topic(), err.Error())
			i.CountStoreError(m.Topic())
			return
		}
		i.CountSuccess(m.Topic())
	}
}
