package config

import (
	"io/ioutil"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v2"
)

const GaugeValueType = "gauge"
const CounterValueType = "counter"

var MQTTConfigDefaults = MQTTConfig{
	Server:    "tcp://127.0.0.1:1883",
	TopicPath: "v1/devices/me",
	QoS:       0,
}

var CacheConfigDefaults = CacheConfig{
	Timeout: 2 * time.Minute,
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
	Server    string `yaml:"server"`
	TopicPath string `yaml:"topic_path"`
	User      string `yaml:"user"`
	Password  string `yaml:"password"`
	QoS       byte   `yaml:"qos"`
}

// Metrics Config is a mapping between a metric send on mqtt to a prometheus metric
type MetricConfig struct {
	PrometheusName     string                    `yaml:"prom_name"`
	MQTTName           string                    `yaml:"mqtt_name"`
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
		mc.PrometheusName, mc.Help, []string{"sensor"}, mc.ConstantLabels,
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
	return cfg, nil
}
