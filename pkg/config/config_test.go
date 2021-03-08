package config

import (
	"reflect"
	"testing"
)

func TestRegexp_GroupValue(t *testing.T) {
	type args struct {
		s         string
		groupName string
	}
	tests := []struct {
		name    string
		pattern string
		args    args
		want    string
	}{
		{
			name:    "normal match",
			pattern: "(.*/)?(?P<deviceid>.*)",
			args: args{
				s:         "foo/bar",
				groupName: "deviceid",
			},
			want: "bar",
		},
		{
			name:    "two groups",
			pattern: "(.*/)?(?P<deviceid>.*)/(?P<service>.*)",
			args: args{
				s:         "foo/bar/batz",
				groupName: "deviceid",
			},
			want: "bar",
		},
		{
			name:    "empty string match",
			pattern: "(.*/)?(?P<deviceid>.*)",
			args: args{
				s:         "",
				groupName: "deviceid",
			},
			want: "",
		},
		{
			name:    "not match",
			pattern: "(.*)/(?P<deviceid>.*)",
			args: args{
				s:         "bar",
				groupName: "deviceid",
			},
			want: "",
		},
		{
			name:    "empty pattern",
			pattern: "",
			args: args{
				s:         "bar",
				groupName: "deviceid",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rf := MustNewRegexp(tt.pattern)
			if got := rf.GroupValue(tt.args.s, tt.args.groupName); got != tt.want {
				t.Errorf("GroupValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegexp_Match(t *testing.T) {
	tests := []struct {
		name  string
		regex *Regexp
		args  string
		want  bool
	}{
		{
			name:  "nil regex matches everything",
			regex: nil,
			args:  "foo",
			want:  true,
		},
		{
			name:  "empty regex matches everything",
			regex: &Regexp{},
			args:  "foo",
			want:  true,
		},
		{
			name:  "regex matches",
			regex: MustNewRegexp(".*"),
			args:  "foo",
			want:  true,
		},

		{
			name:  "regex matches",
			regex: MustNewRegexp("a.*"),
			args:  "foo",
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rf := tt.regex
			if got := rf.Match(tt.args); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegexp_MarshalYAML(t *testing.T) {
	tests := []struct {
		name    string
		regex   *Regexp
		want    interface{}
		wantErr bool
	}{
		{
			name:    "empty",
			regex:   nil,
			want:    "",
			wantErr: false,
		},
		{
			name:    "with pattern",
			regex:   MustNewRegexp("a.*"),
			want:    "a.*",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.regex.MarshalYAML()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalYAML() got = %v, want %v", got, tt.want)
			}
		})
	}
}
