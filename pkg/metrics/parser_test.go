package metrics

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/hikhvar/mqtt2prometheus/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
)

func TestParser_parseMetric(t *testing.T) {
	stateDir, err := os.MkdirTemp("", "parser_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(stateDir)

	now = testNow
	type fields struct {
		metricConfigs map[string][]config.MetricConfig
	}
	type args struct {
		metricPath string
		deviceID   string
		value      interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Metric
		wantErr bool
	}{
		{
			name: "value without timestamp",
			fields: fields{
				map[string][]config.MetricConfig{
					"temperature": []config.MetricConfig{
						{
							PrometheusName: "temperature",
							ValueType:      "gauge",
							OmitTimestamp:  true,
						},
					},
				},
			},
			args: args{
				metricPath: "temperature",
				deviceID:   "dht22",
				value:      12.6,
			},
			want: Metric{
				Description: prometheus.NewDesc("temperature", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       12.6,
				IngestTime:  time.Time{},
				Topic:       "",
			},
		},
		{
			name: "string value",
			fields: fields{
				map[string][]config.MetricConfig{
					"temperature": []config.MetricConfig{
						{
							PrometheusName: "temperature",
							ValueType:      "gauge",
						},
					},
				},
			},
			args: args{
				metricPath: "temperature",
				deviceID:   "dht22",
				value:      "12.6",
			},
			want: Metric{
				Description: prometheus.NewDesc("temperature", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       12.6,
				IngestTime:  testNow(),
				Topic:       "",
			},
		},
		{
			name: "scaled string value",
			fields: fields{
				map[string][]config.MetricConfig{
					"temperature": []config.MetricConfig{
						{
							PrometheusName: "temperature",
							ValueType:      "gauge",
							MQTTValueScale: 0.01,
						},
					},
				},
			},
			args: args{
				metricPath: "temperature",
				deviceID:   "dht22",
				value:      "12.6",
			},
			want: Metric{
				Description: prometheus.NewDesc("temperature", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       0.126,
				IngestTime:  testNow(),
				Topic:       "",
			},
		},
		{
			name: "string value failure",
			fields: fields{
				map[string][]config.MetricConfig{
					"temperature": []config.MetricConfig{
						{
							PrometheusName: "temperature",
							ValueType:      "gauge",
						},
					},
				},
			},
			args: args{
				metricPath: "temperature",
				deviceID:   "dht22",
				value:      "12.6.5",
			},
			wantErr: true,
		},
		{
			name: "float value",
			fields: fields{
				map[string][]config.MetricConfig{
					"temperature": []config.MetricConfig{
						{
							PrometheusName: "temperature",
							ValueType:      "gauge",
						},
					},
				},
			},
			args: args{
				metricPath: "temperature",
				deviceID:   "dht22",
				value:      12.6,
			},
			want: Metric{
				Description: prometheus.NewDesc("temperature", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       12.6,
				IngestTime:  testNow(),
				Topic:       "",
			},
		},
		{
			name: "scaled float value",
			fields: fields{
				map[string][]config.MetricConfig{
					"humidity": []config.MetricConfig{
						{
							PrometheusName: "humidity",
							ValueType:      "gauge",
							MQTTValueScale: 0.01,
						},
					},
				},
			},
			args: args{
				metricPath: "humidity",
				deviceID:   "dht22",
				value:      12.6,
			},
			want: Metric{
				Description: prometheus.NewDesc("humidity", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       0.126,
				IngestTime:  testNow(),
				Topic:       "",
			},
		},
		{
			name: "negative scaled float value",
			fields: fields{
				map[string][]config.MetricConfig{
					"humidity": []config.MetricConfig{
						{
							PrometheusName: "humidity",
							ValueType:      "gauge",
							MQTTValueScale: -2,
						},
					},
				},
			},
			args: args{
				metricPath: "humidity",
				deviceID:   "dht22",
				value:      12.6,
			},
			want: Metric{
				Description: prometheus.NewDesc("humidity", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       -25.2,
				IngestTime:  testNow(),
				Topic:       "",
			},
		},
		{
			name: "bool value true",
			fields: fields{
				map[string][]config.MetricConfig{
					"enabled": []config.MetricConfig{
						{
							PrometheusName: "enabled",
							ValueType:      "gauge",
						},
					},
				},
			},
			args: args{
				metricPath: "enabled",
				deviceID:   "dht22",
				value:      true,
			},
			want: Metric{
				Description: prometheus.NewDesc("enabled", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       1,
				IngestTime:  testNow(),
				Topic:       "",
			},
		},
		{
			name: "scaled bool value",
			fields: fields{
				map[string][]config.MetricConfig{
					"enabled": []config.MetricConfig{
						{
							PrometheusName: "enabled",
							ValueType:      "gauge",
							MQTTValueScale: 0.5,
						},
					},
				},
			},
			args: args{
				metricPath: "enabled",
				deviceID:   "dht22",
				value:      true,
			},
			want: Metric{
				Description: prometheus.NewDesc("enabled", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       0.5,
				IngestTime:  testNow(),
				Topic:       "",
			},
		},
		{
			name: "bool value false",
			fields: fields{
				map[string][]config.MetricConfig{
					"enabled": []config.MetricConfig{
						{
							PrometheusName: "enabled",
							ValueType:      "gauge",
						},
					},
				},
			},
			args: args{
				metricPath: "enabled",
				deviceID:   "dht22",
				value:      false,
			},
			want: Metric{
				Description: prometheus.NewDesc("enabled", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       0,
				IngestTime:  testNow(),
				Topic:       "",
			},
		},
		{
			name: "string mapping value success",
			fields: fields{
				map[string][]config.MetricConfig{
					"enabled": []config.MetricConfig{
						{
							PrometheusName: "enabled",
							ValueType:      "gauge",
							StringValueMapping: &config.StringValueMappingConfig{
								Map: map[string]float64{
									"foo": 112,
									"bar": 2,
								},
							},
						},
					},
				},
			},
			args: args{
				metricPath: "enabled",
				deviceID:   "dht22",
				value:      "foo",
			},
			want: Metric{
				Description: prometheus.NewDesc("enabled", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       112,
				IngestTime:  testNow(),
				Topic:       "",
			},
		},
		{
			name: "string mapping value failure default to error value",
			fields: fields{
				map[string][]config.MetricConfig{
					"enabled": []config.MetricConfig{
						{
							PrometheusName: "enabled",
							ValueType:      "gauge",
							StringValueMapping: &config.StringValueMappingConfig{
								ErrorValue: floatP(12333),
								Map: map[string]float64{
									"foo": 112,
									"bar": 2,
								},
							},
						},
					},
				},
			},
			args: args{
				metricPath: "enabled",
				deviceID:   "dht22",
				value:      "asd",
			},
			want: Metric{
				Description: prometheus.NewDesc("enabled", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       12333,
				IngestTime:  testNow(),
				Topic:       "",
			},
		},
		{
			name: "string mapping value failure no error value",
			fields: fields{
				map[string][]config.MetricConfig{
					"enabled": []config.MetricConfig{
						{
							PrometheusName: "enabled",
							ValueType:      "gauge",
							StringValueMapping: &config.StringValueMappingConfig{
								Map: map[string]float64{
									"foo": 112,
									"bar": 2,
								},
							},
						},
					},
				},
			},
			args: args{
				metricPath: "enabled",
				deviceID:   "dht22",
				value:      "asd",
			},
			wantErr: true,
		},
		{
			name: "metric not configured",
			fields: fields{
				map[string][]config.MetricConfig{
					"enabled": []config.MetricConfig{
						{
							PrometheusName: "enabled",
							ValueType:      "gauge",
							StringValueMapping: &config.StringValueMappingConfig{
								ErrorValue: floatP(12333),
								Map: map[string]float64{
									"foo": 112,
									"bar": 2,
								},
							},
						},
					},
				},
			},
			args: args{
				metricPath: "enabled1",
				deviceID:   "dht22",
				value:      "asd",
			},
			wantErr: true,
		},
		{
			name: "unexpected type",
			fields: fields{
				map[string][]config.MetricConfig{
					"enabled": []config.MetricConfig{
						{
							PrometheusName: "enabled",
							ValueType:      "gauge",
							StringValueMapping: &config.StringValueMappingConfig{
								ErrorValue: floatP(12333),
								Map: map[string]float64{
									"foo": 112,
									"bar": 2,
								},
							},
						},
					},
				},
			},
			args: args{
				metricPath: "enabled",
				deviceID:   "dht22",
				value:      []int{3},
			},
			wantErr: true,
		},
		{
			name: "monotonic gauge, step 1: initial value",
			fields: fields{
				map[string][]config.MetricConfig{
					"aenergy.total": []config.MetricConfig{
						{
							PrometheusName:  "total_energy",
							ValueType:       "gauge",
							OmitTimestamp:   true,
							ForceMonotonicy: true,
						},
					},
				},
			},
			args: args{
				metricPath: "aenergy.total",
				deviceID:   "shellyplus1pm-foo",
				value:      1.0,
			},
			want: Metric{
				Description: prometheus.NewDesc("total_energy", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       1.0,
			},
		},
		{
			name: "monotonic gauge, step 2: monotonic increase does not add offset",
			fields: fields{
				map[string][]config.MetricConfig{
					"aenergy.total": []config.MetricConfig{
						{
							PrometheusName:  "total_energy",
							ValueType:       "gauge",
							OmitTimestamp:   true,
							ForceMonotonicy: true,
						},
					},
				},
			},
			args: args{
				metricPath: "aenergy.total",
				deviceID:   "shellyplus1pm-foo",
				value:      2.0,
			},
			want: Metric{
				Description: prometheus.NewDesc("total_energy", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       2.0,
			},
		},
		{
			name: "monotonic gauge, step 3: raw metric is reset, last value becomes the new offset",
			fields: fields{
				map[string][]config.MetricConfig{
					"aenergy.total": []config.MetricConfig{
						{
							PrometheusName:  "total_energy",
							ValueType:       "gauge",
							OmitTimestamp:   true,
							ForceMonotonicy: true,
						},
					},
				},
			},
			args: args{
				metricPath: "aenergy.total",
				deviceID:   "shellyplus1pm-foo",
				value:      0.0,
			},
			want: Metric{
				Description: prometheus.NewDesc("total_energy", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       2.0,
			},
		},
		{
			name: "monotonic gauge, step 4: monotonic increase with offset",
			fields: fields{
				map[string][]config.MetricConfig{
					"aenergy.total": []config.MetricConfig{
						{
							PrometheusName:  "total_energy",
							ValueType:       "gauge",
							OmitTimestamp:   true,
							ForceMonotonicy: true,
						},
					},
				},
			},
			args: args{
				metricPath: "aenergy.total",
				deviceID:   "shellyplus1pm-foo",
				value:      1.0,
			},
			want: Metric{
				Description: prometheus.NewDesc("total_energy", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       3.0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(nil, config.JsonParsingConfigDefaults.Separator, stateDir)
			p.metricConfigs = tt.fields.metricConfigs

			// Find a valid metrics config
			config, found := p.findMetricConfig(tt.args.metricPath, tt.args.deviceID)
			if !found {
				if !tt.wantErr {
					t.Errorf("MetricConfig not found")
				}
				return
			}

			id := metricID("", tt.args.metricPath, tt.args.deviceID, config.PrometheusName)
			got, err := p.parseMetric(config, id, tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMetric() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseMetric() got = %v, want %v", got, tt.want)
			}

			if config.ForceMonotonicy {
				if err = p.writeMetricState(id, p.states[id]); err != nil {
					t.Errorf("failed to write metric state: %v", err)
				}
			}
		})
	}
}

func testNow() time.Time {
	now, err := time.Parse(
		time.RFC3339,
		"2020-11-01T22:08:41+00:00")
	if err != nil {
		panic(err)
	}
	return now
}

func floatP(f float64) *float64 {
	return &f
}
