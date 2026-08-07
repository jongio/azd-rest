package config

import (
	"reflect"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	want := Config{
		DataFormat:      "json",
		OutputFormat:    "auto",
		Retry:           3,
		Timeout:         30 * time.Second,
		FollowRedirects: true,
		MaxRedirects:    10,
		MaxPages:        100,
		MaxResponseSize: 100 * 1024 * 1024,
		Repeat:          1,
		Color:           "auto",
	}

	if got := Defaults(); !reflect.DeepEqual(got, want) {
		t.Errorf("Defaults() = %#v, want %#v", got, want)
	}
}
