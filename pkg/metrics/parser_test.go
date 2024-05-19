package metrics

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/hikhvar/mqtt2prometheus/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
)

var testNowElapsed time.Duration

func testNow() time.Time {
	now, err := time.Parse(
		time.RFC3339,
		"2020-11-01T22:08:41+00:00")
	if err != nil {
		panic(err)
	}
	now = now.Add(testNowElapsed)
	return now
}

func TestParser_parseMetric(t *testing.T) {
	stateDir, err := os.MkdirTemp("", "parser_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(stateDir)

	now = testNow
	type fields struct {
		metricConfigs map[string][]*config.MetricConfig
	}
	type args struct {
		metricPath string
		deviceID   string
		value      interface{}
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      Metric
		wantErr   bool
		elapseNow time.Duration
	}{
		{
			name: "value without timestamp",
			fields: fields{
				map[string][]*config.MetricConfig{
					"temperature": {
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
				map[string][]*config.MetricConfig{
					"temperature": {
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
				map[string][]*config.MetricConfig{
					"temperature": {
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
				map[string][]*config.MetricConfig{
					"temperature": {
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
				map[string][]*config.MetricConfig{
					"temperature": {
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
				map[string][]*config.MetricConfig{
					"humidity": {
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
				map[string][]*config.MetricConfig{
					"humidity": {
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
				map[string][]*config.MetricConfig{
					"enabled": {
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
				map[string][]*config.MetricConfig{
					"enabled": {
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
				map[string][]*config.MetricConfig{
					"enabled": {
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
				map[string][]*config.MetricConfig{
					"enabled": {
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
				map[string][]*config.MetricConfig{
					"enabled": {
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
				map[string][]*config.MetricConfig{
					"enabled": {
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
				map[string][]*config.MetricConfig{
					"enabled": {
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
				map[string][]*config.MetricConfig{
					"enabled": {
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
				map[string][]*config.MetricConfig{
					"aenergy.total": {
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
				map[string][]*config.MetricConfig{
					"aenergy.total": {
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
				map[string][]*config.MetricConfig{
					"aenergy.total": {
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
				map[string][]*config.MetricConfig{
					"aenergy.total": {
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
		{
			name: "integrate positive values using expressions, step 1",
			fields: fields{
				map[string][]*config.MetricConfig{
					"apower": {
						{
							PrometheusName: "total_energy",
							ValueType:      "gauge",
							OmitTimestamp:  true,
							Expression:     "value > 0 ? last_result + value * elapsed.Hours() : last_result",
						},
					},
				},
			},
			args: args{
				metricPath: "apower",
				deviceID:   "shellyplus1pm-foo",
				value:      60.0,
			},
			want: Metric{
				Description: prometheus.NewDesc("total_energy", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       0.0, // No elapsed time yet, hence no integration
			},
		},
		{
			name: "integrate positive values using expressions, step 2",
			fields: fields{
				map[string][]*config.MetricConfig{
					"apower": {
						{
							PrometheusName: "total_energy",
							ValueType:      "gauge",
							OmitTimestamp:  true,
							Expression:     "value > 0 ? last_result + value * elapsed.Hours() : last_result",
						},
					},
				},
			},
			elapseNow: time.Minute,
			args: args{
				metricPath: "apower",
				deviceID:   "shellyplus1pm-foo",
				value:      60.0,
			},
			want: Metric{
				Description: prometheus.NewDesc("total_energy", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       1.0, // 60 watts for 1 minute = 1 Wh
			},
		},
		{
			name: "integrate positive values using expressions, step 3",
			fields: fields{
				map[string][]*config.MetricConfig{
					"apower": {
						{
							PrometheusName: "total_energy",
							ValueType:      "gauge",
							OmitTimestamp:  true,
							Expression:     "value > 0 ? last_result + value * elapsed.Hours() : last_result",
						},
					},
				},
			},
			elapseNow: 2 * time.Minute,
			args: args{
				metricPath: "apower",
				deviceID:   "shellyplus1pm-foo",
				value:      -60.0,
			},
			want: Metric{
				Description: prometheus.NewDesc("total_energy", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       1.0, // negative input is ignored
			},
		},
		{
			name: "integrate positive values using expressions, step 4",
			fields: fields{
				map[string][]*config.MetricConfig{
					"apower": {
						{
							PrometheusName: "total_energy",
							ValueType:      "gauge",
							OmitTimestamp:  true,
							Expression:     "value > 0 ? last_result + value * elapsed.Hours() : last_result",
						},
					},
				},
			},
			elapseNow: 3 * time.Minute,
			args: args{
				metricPath: "apower",
				deviceID:   "shellyplus1pm-foo",
				value:      600.0,
			},
			want: Metric{
				Description: prometheus.NewDesc("total_energy", "", []string{"sensor", "topic"}, nil),
				ValueType:   prometheus.GaugeValue,
				Value:       11.0, // 600 watts for 1 minute = 10 Wh
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testNowElapsed = tt.elapseNow
			defer func() { testNowElapsed = time.Duration(0) }()

			p := NewParser(nil, config.JsonParsingConfigDefaults.Separator, stateDir)
			p.metricConfigs = tt.fields.metricConfigs

			// Find a valid metrics config
			configs := p.findMetricConfigs(tt.args.metricPath, tt.args.deviceID)
			if len(configs) != 1 {
				if !tt.wantErr {
					t.Errorf("MetricConfig not found")
				}
				return
			}
			config := configs[0]

			id := metricID("", tt.args.metricPath, tt.args.deviceID, config.PrometheusName)
			got, err := p.parseMetric(config, id, tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMetric() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseMetric() got = %v, want %v", got, tt.want)
			}

			if config.ForceMonotonicy || config.Expression != "" {
				if err = p.writeMetricState(id, p.states[id]); err != nil {
					t.Errorf("failed to write metric state: %v", err)
				}
			}
		})
	}
}

func floatP(f float64) *float64 {
	return &f
}

func TestParser_evalExpression(t *testing.T) {
	now = testNow
	testNowElapsed = time.Duration(0)
	id := "metric"

	tests := []struct {
		expression string
		values     []float64
		results    []float64
	}{
		{
			expression: "value + value",
			values:     []float64{1, 0, -4},
			results:    []float64{2, 0, -8},
		},
		{
			expression: "value - last_value",
			values:     []float64{1, 2, 5, 7},
			results:    []float64{1, 1, 3, 2},
		},
		{
			expression: "last_result + value",
			values:     []float64{1, 2, 3, 4},
			results:    []float64{1, 3, 6, 10},
		},
		{
			expression: "last_result + elapsed.Milliseconds()",
			values:     []float64{0, 0, 0, 0},
			results:    []float64{0, 1000, 2000, 3000},
		},
		{
			expression: "now().Unix()",
			values:     []float64{0, 0},
			results:    []float64{float64(testNow().Unix()), float64(testNow().Unix() + 1)},
		},
		{
			expression: "int(1.1) + int(1.9)",
			values:     []float64{0},
			results:    []float64{2},
		},
		{
			expression: "float(elapsed)",
			values:     []float64{0, 0},
			results:    []float64{0, float64(time.Second)},
		},
		{
			expression: "round(value)",
			values:     []float64{1.1, 2.5, 3.9},
			results:    []float64{1, 3, 4},
		},
		{
			expression: "ceil(value)",
			values:     []float64{1.1, 2.9, 4.0},
			results:    []float64{2, 3, 4},
		},
		{
			expression: "floor(value)",
			values:     []float64{1.1, 2.9, 4.0},
			results:    []float64{1, 2, 4},
		},
		{
			expression: "abs(value)",
			values:     []float64{0, 1, -2},
			results:    []float64{0, 1, 2},
		},
		{
			expression: "min(value, 0)",
			values:     []float64{1, -2, 3, -4},
			results:    []float64{0, -2, 0, -4},
		},
		{
			expression: "max(value, 0)",
			values:     []float64{1, -2, 3, -4},
			results:    []float64{1, 0, 3, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.expression, func(t *testing.T) {
			stateDir, err := os.MkdirTemp("", "parser_test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(stateDir)
			defer func() { testNowElapsed = time.Duration(0) }()

			p := NewParser(nil, ".", stateDir)
			for i, value := range tt.values {
				got, err := p.evalExpression(id, tt.expression, value)
				want := tt.results[i]
				if err != nil {
					t.Errorf("evaluating the %dth value '%v' failed: %v", i, value, err)
				}
				if got != want {
					t.Errorf("unexpected result for %dth value, got %v, want %v", i, got, want)
				}
				// Advance the clock by one second for every sample
				testNowElapsed = testNowElapsed + time.Second
			}
		})
	}
}
