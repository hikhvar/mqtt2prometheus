package metrics

import (
	"fmt"
	"github.com/eclipse/paho.mqtt.golang"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/hikhvar/mqtt2prometheus/pkg/config"
)

type Ingest struct {
	instrumentation
	extractor     Extractor
	deviceIDRegex *config.Regexp
	collector     Collector
	logger        log.Logger
}

func NewIngest(collector Collector, extractor Extractor, deviceIDRegex *config.Regexp, logger log.Logger) *Ingest {

	return &Ingest{
		instrumentation: defaultInstrumentation,
		extractor:       extractor,
		deviceIDRegex:   deviceIDRegex,
		collector:       collector,
		logger:          logger,
	}
}

func (i *Ingest) store(topic string, payload []byte) error {
	deviceID := i.deviceID(topic)
	mc, err := i.extractor(topic, payload, deviceID)
	if err != nil {
		return fmt.Errorf("failed to extract metric values from topic: %w", err)
	}
	i.collector.Observe(deviceID, mc)
	return nil
}

func (i *Ingest) SetupSubscriptionHandler(errChan chan<- error) mqtt.MessageHandler {
	return func(c mqtt.Client, m mqtt.Message) {
		level.Debug(i.logger).Log("msg", "Got message", "topic", m.Topic(), "payload", string(m.Payload()))
		err := i.store(m.Topic(), m.Payload())
		if err != nil {
			errChan <- fmt.Errorf("could not store metrics '%s' on topic %s: %s", string(m.Payload()), m.Topic(), err.Error())
			i.CountStoreError(m.Topic())
			return
		}
		i.CountSuccess(m.Topic())
	}
}

// deviceID uses the configured DeviceIDRegex to extract the device ID from the given mqtt topic path.
func (i *Ingest) deviceID(topic string) string {
	return i.deviceIDRegex.GroupValue(topic, config.DeviceIDRegexGroup)
}
