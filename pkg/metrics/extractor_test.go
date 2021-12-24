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
			if len(got) != 1 {
				t.Errorf("parseMetric() got = %v, want %v", nil, tt.want)
			} else if !reflect.DeepEqual(got[0], tt.want) {
				t.Errorf("parseMetric() got = %v, want %v", got[0], tt.want)
			}
		})
	}
}
