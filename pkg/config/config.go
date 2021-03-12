package config

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v2"
)

const (
	GaugeValueType   = "gauge"
	CounterValueType = "counter"

	DeviceIDRegexGroup   = "deviceid"
	MetricNameRegexGroup = "metricname"
)

var MQTTConfigDefaults = MQTTConfig{
	Server:        "tcp://127.0.0.1:1883",
	TopicPath:     "v1/devices/me",
	DeviceIDRegex: MustNewRegexp(fmt.Sprintf("(.*/)?(?P<%s>.*)", DeviceIDRegexGroup)),
	QoS:           0,
}

var CacheConfigDefaults = CacheConfig{
	Timeout: 2 * time.Minute,
}

type Regexp struct {
	r       *regexp.Regexp
	pattern string
}

func (rf *Regexp) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var pattern string
	if err := unmarshal(&pattern); err != nil {
		return err
	}
	r, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	rf.r = r
	rf.pattern = pattern
	return nil
}

func (rf *Regexp) MarshalYAML() (interface{}, error) {
	if rf == nil {
		return "", nil
	}
	return rf.pattern, nil
}

func (rf *Regexp) Match(s string) bool {
	return rf == nil || rf.r == nil || rf.r.MatchString(s)
}

// GroupValue returns the value of the given group. If the group is not part of the underlying regexp, returns the empty string.
func (rf *Regexp) GroupValue(s string, groupName string) string {
	match := rf.r.FindStringSubmatch(s)
	groupValues := make(map[string]string)
	for i, name := range rf.r.SubexpNames() {
		if len(match) > i && name != "" {
			groupValues[name] = match[i]
		}
	}
	return groupValues[groupName]
}

func (rf *Regexp) RegEx() *regexp.Regexp {
	return rf.r
}

func MustNewRegexp(pattern string) *Regexp {
	return &Regexp{
		pattern: pattern,
		r:       regexp.MustCompile(pattern),
	}
}

type Config struct {
	Metrics []MetricConfig `yaml:"metrics"`
	MQTT    *MQTTConfig    `yaml:"mqtt,omitempty"`
	Cache   *CacheConfig   `yaml:"cache,omitempty"`
}

type CacheConfig struct {
	Timeout time.Duration `yaml:"timeout"`
}

type MQTTConfig struct {
	Server               string                `yaml:"server"`
	TopicPath            string                `yaml:"topic_path"`
	DeviceIDRegex        *Regexp               `yaml:"device_id_regex"`
	User                 string                `yaml:"user"`
	Password             string                `yaml:"password"`
	QoS                  byte                  `yaml:"qos"`
	ObjectPerTopicConfig *ObjectPerTopicConfig `yaml:"object_per_topic_config"`
	MetricPerTopicConfig *MetricPerTopicConfig `yaml:"metric_per_topic_config"`
	CACert               string                `yaml:"ca_cert"`
	ClientCert           string                `yaml:"client_cert"`
	ClientKey            string                `yaml:"client_key"`
	ClientID             string                `yaml:"client_id"`
}

const EncodingJSON = "JSON"

type ObjectPerTopicConfig struct {
	Encoding string `yaml:"encoding"` // Currently only JSON is a valid value
}

type MetricPerTopicConfig struct {
	MetricNameRegex *Regexp `yaml:"metric_name_regex"` // Default
}

// Metrics Config is a mapping between a metric send on mqtt to a prometheus metric
type MetricConfig struct {
	PrometheusName     string                    `yaml:"prom_name"`
	MQTTName           string                    `yaml:"mqtt_name"`
	SensorNameFilter   Regexp                    `yaml:"sensor_name_filter"`
	Help               string                    `yaml:"help"`
	ValueType          string                    `yaml:"type"`
	ConstantLabels     map[string]string         `yaml:"const_labels"`
	StringValueMapping *StringValueMappingConfig `yaml:"string_value_mapping"`
}

// StringValueMappingConfig defines the mapping from string to float
type StringValueMappingConfig struct {
	// ErrorValue is used when no mapping is found in Map
	ErrorValue *float64           `yaml:"error_value"`
	Map        map[string]float64 `yaml:"map"`
}

func (mc *MetricConfig) PrometheusDescription() *prometheus.Desc {
	return prometheus.NewDesc(
		mc.PrometheusName, mc.Help, []string{"sensor", "topic"}, mc.ConstantLabels,
	)
}

func (mc *MetricConfig) PrometheusValueType() prometheus.ValueType {
	switch mc.ValueType {
	case GaugeValueType:
		return prometheus.GaugeValue
	case CounterValueType:
		return prometheus.CounterValue
	default:
		return prometheus.UntypedValue
	}
}

func LoadConfig(configFile string) (Config, error) {
	configData, err := ioutil.ReadFile(configFile)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err = yaml.Unmarshal(configData, &cfg); err != nil {
		return cfg, err
	}
	if cfg.MQTT == nil {
		cfg.MQTT = &MQTTConfigDefaults
	}
	if cfg.Cache == nil {
		cfg.Cache = &CacheConfigDefaults
	}
	if cfg.MQTT.DeviceIDRegex == nil {
		cfg.MQTT.DeviceIDRegex = MQTTConfigDefaults.DeviceIDRegex
	}
	var validRegex bool
	for _, name := range cfg.MQTT.DeviceIDRegex.RegEx().SubexpNames() {
		if name == DeviceIDRegexGroup {
			validRegex = true
		}
	}
	if !validRegex {
		return Config{}, fmt.Errorf("device id regex %q does not contain required regex group %q", cfg.MQTT.DeviceIDRegex.pattern, DeviceIDRegexGroup)
	}
	if cfg.MQTT.ObjectPerTopicConfig == nil && cfg.MQTT.MetricPerTopicConfig == nil {
		cfg.MQTT.ObjectPerTopicConfig = &ObjectPerTopicConfig{
			Encoding: EncodingJSON,
		}
	}

	if cfg.MQTT.MetricPerTopicConfig != nil {
		validRegex = false
		for _, name := range cfg.MQTT.MetricPerTopicConfig.MetricNameRegex.RegEx().SubexpNames() {
			if name == MetricNameRegexGroup {
				validRegex = true
			}
		}
		if !validRegex {
			return Config{}, fmt.Errorf("metric name regex %q does not contain required regex group %q", cfg.MQTT.DeviceIDRegex.pattern, MetricNameRegexGroup)
		}
	}

	return cfg, nil
}
