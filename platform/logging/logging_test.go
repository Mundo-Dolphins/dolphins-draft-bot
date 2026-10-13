package logging

import "testing"

func TestNewLoggerLevels(t *testing.T) {
	levels := []string{"debug", "warn", "error", "info", "other"}
	for _, l := range levels {
		if New(l, "json") == nil {
			t.Fatalf("nil logger for %s", l)
		}
	}
}

func TestNewLoggerFormats(t *testing.T) {
	formats := []string{"json", "text", "keyvalue", "console", "other"}
	for _, format := range formats {
		if New("info", format) == nil {
			t.Fatalf("nil logger for format %s", format)
		}
	}
}
