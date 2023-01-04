package metrics

import (
	"reflect"
	"testing"

	"github.com/hikhvar/mqtt2prometheus/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewJSONObjectExtractor_parseMetric(t *testing.T) {
	now = testNow
	type fields struct {
		metricConfigs map[string][]config.MetricConfig
	}
	type args struct {
		metricPath string
		deviceID   string
		value      string
	}
	tests := []struct {
		name      string
		separator string
		fields    fields
		args      args
		want      Metric
		wantErr   bool
		noValue   bool
	}{
		{
			name:      "string value",
			separator: "->",
			fields: fields{
				map[string][]config.MetricConfig{
					"SDS0X1->PM2->5": []config.MetricConfig{
						{
							PrometheusName: "temperature",
							MQTTName:       "SDS0X1.PM2.5",
							ValueType:      "gauge",
						},
					},
				},
			},
			args: args{
				metricPath: "topic",
				deviceID:   "dht22",
				value:      "{\"SDS0X1\":{\"PM2\":{\"5\":4.9}}}",
			},
			want: Metric{
				Description: prometheus.NewDesc("temperature", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       4.9,
				IngestTime:  testNow(),
				Topic:       "topic",
			},
		}, {
			name:      "string value with dots in path",
			separator: "->",
			fields: fields{
				map[string][]config.MetricConfig{
					"SDS0X1->PM2.5": []config.MetricConfig{
						{
							PrometheusName: "temperature",
							MQTTName:       "SDS0X1->PM2.5",
							ValueType:      "gauge",
						},
					},
				},
			},
			args: args{
				metricPath: "topic",
				deviceID:   "dht22",
				value:      "{\"SDS0X1\":{\"PM2.5\":4.9,\"PM10\":8.5}}",
			},
			want: Metric{
				Description: prometheus.NewDesc("temperature", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       4.9,
				IngestTime:  testNow(),
				Topic:       "topic",
			},
		}, {
			name:      "metric matching SensorNameFilter",
			separator: ".",
			fields: fields{
				map[string][]config.MetricConfig{
					"temperature": []config.MetricConfig{
						{
							PrometheusName:   "temperature",
							MQTTName:         "temperature",
							ValueType:        "gauge",
							SensorNameFilter: *config.MustNewRegexp(".*22$"),
						},
					},
				},
			},
			args: args{
				metricPath: "topic",
				deviceID:   "dht22",
				value:      "{\"temperature\": 8.5}",
			},
			want: Metric{
				Description: prometheus.NewDesc("temperature", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       8.5,
				IngestTime:  testNow(),
				Topic:       "topic",
			},
		}, {
			name:      "metric not matching SensorNameFilter",
			separator: ".",
			fields: fields{
				map[string][]config.MetricConfig{
					"temperature": []config.MetricConfig{
						{
							PrometheusName:   "temperature",
							MQTTName:         "temperature",
							ValueType:        "gauge",
							SensorNameFilter: *config.MustNewRegexp(".*fail$"),
						},
					},
				},
			},
			args: args{
				metricPath: "topic",
				deviceID:   "dht22",
				value:      "{\"temperature\": 8.5}",
			},
			want:    Metric{},
			noValue: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Parser{
				separator:     tt.separator,
				metricConfigs: tt.fields.metricConfigs,
			}
			extractor := NewJSONObjectExtractor(p)

			got, err := extractor(tt.args.metricPath, []byte(tt.args.value), tt.args.deviceID)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMetric() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) == 0 {
				if !tt.noValue {
					t.Errorf("parseMetric() got = %v, want %v", nil, tt.want)
				}
			} else if !reflect.DeepEqual(got[0], tt.want) {
				t.Errorf("parseMetric() got = %v, want %v", got[0], tt.want)
			} else if len(got) > 1 {
				t.Errorf("unexpected result got = %v, want %v", got, tt.want)
			}
		})
	}
}
