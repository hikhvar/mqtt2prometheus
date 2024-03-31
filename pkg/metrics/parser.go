package metrics

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hikhvar/mqtt2prometheus/pkg/config"
	"gopkg.in/yaml.v2"
)

// monotonicState holds the runtime information to realize a monotonic increasing value.
type monotonicState struct {
	// Basline value to add to each parsed metric value to maintain monotonicy
	Offset float64 `yaml:"value_offset"`
	// Last value that was parsed before the offset was added
	LastRawValue float64 `yaml:"last_raw_value"`
}

// metricState holds runtime information per metric configuration.
type metricState struct {
	monotonic monotonicState
	// The last time the state file was written
	lastWritten time.Time
}

type Parser struct {
	separator string
	// Maps the mqtt metric name to a list of configs
	// The first that matches SensorNameFilter will be used
	metricConfigs map[string][]config.MetricConfig
	// Directory holding state files
	stateDir string
	// Per-metric state
	states map[string]*metricState
}

var now = time.Now

func NewParser(metrics []config.MetricConfig, separator, stateDir string) Parser {
	cfgs := make(map[string][]config.MetricConfig)
	for i := range metrics {
		key := metrics[i].MQTTName
		cfgs[key] = append(cfgs[key], metrics[i])
	}
	return Parser{
		separator:     separator,
		metricConfigs: cfgs,
		stateDir:      strings.TrimRight(stateDir, "/"),
		states:        make(map[string]*metricState),
	}
}

// Config returns the underlying metrics config
func (p *Parser) config() map[string][]config.MetricConfig {
	return p.metricConfigs
}

// validMetric returns config matching the metric and deviceID
// Second return value indicates if config was found.
func (p *Parser) findMetricConfig(metric string, deviceID string) (config.MetricConfig, bool) {
	for _, c := range p.metricConfigs[metric] {
		if c.SensorNameFilter.Match(deviceID) {
			return c, true
		}
	}
	return config.MetricConfig{}, false
}

// parseMetric parses the given value according to the given deviceID and metricPath. The config allows to
// parse a metric value according to the device ID.
func (p *Parser) parseMetric(cfg config.MetricConfig, metricID string, value interface{}) (Metric, error) {
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

	if cfg.ForceMonotonicy {
		ms, err := p.getMetricState(metricID)
		if err != nil {
			return Metric{}, err
		}
		// When the source metric is reset, the last adjusted value becomes the new offset.
		if metricValue < ms.monotonic.LastRawValue {
			ms.monotonic.Offset += ms.monotonic.LastRawValue
			// Trigger flushing the new state to disk.
			ms.lastWritten = time.Time{}
		}

		ms.monotonic.LastRawValue = metricValue
		metricValue += ms.monotonic.Offset
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

func (p *Parser) stateFileName(metricID string) string {
	return fmt.Sprintf("%s/%s.yaml", p.stateDir, metricID)
}

// readMetricState parses the metric state from the configured path.
// If the file does not exist, an empty state is returned.
func (p *Parser) readMetricState(metricID string) (*metricState, error) {
	state := &metricState{}
	f, err := os.Open(p.stateFileName(metricID))
	if err != nil {
		// The file does not exist for new metrics.
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, err
	}
	defer f.Close()

	var data []byte
	if info, err := f.Stat(); err == nil {
		data = make([]byte, int(info.Size()))
	}
	if _, err := f.Read(data); err != nil && err != io.EOF {
		return state, err
	}

	err = yaml.UnmarshalStrict(data, &state.monotonic)
	state.lastWritten = now()
	return state, err
}

// writeMetricState writes back the metric's current state to the configured path.
func (p *Parser) writeMetricState(metricID string, state *metricState) error {
	out, err := yaml.Marshal(state.monotonic)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(p.stateFileName(metricID), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(out)
	f.Close()
	return err
}

// getMetricState returns the state of the given metric.
// The state is read from and written back to disk as needed.
func (p *Parser) getMetricState(metricID string) (*metricState, error) {
	var err error
	state, found := p.states[metricID]
	if !found {
		if state, err = p.readMetricState(metricID); err != nil {
			return nil, err
		}
		p.states[metricID] = state
	}
	// Write the state back to disc every minute.
	if now().Sub(state.lastWritten) >= time.Minute {
		if err = p.writeMetricState(metricID, state); err == nil {
			state.lastWritten = now()
		}
	}
	return state, err
}
