package metrics

import (
	"reflect"
	"testing"
	"time"

	"github.com/hikhvar/mqtt2prometheus/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
)

func TestParser_parseMetric(t *testing.T) {
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
				value:      metricNotConfigured,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Parser{
				metricConfigs: tt.fields.metricConfigs,
			}
			got, err := p.parseMetric(tt.args.metricPath, tt.args.deviceID, tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMetric() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseMetric() got = %v, want %v", got, tt.want)
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
