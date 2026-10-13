package runner

import (
	"testing"
	"time"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/platform/config"
)

func TestBuildWindowsTimezone(t *testing.T) {
	cfg := config.Config{Windows: []config.WindowConfig{{Name: "r1", Start: "2026-04-24T02:00:00", Rounds: []int{1}}}}
	loc, _ := time.LoadLocation("Europe/Madrid")
	w, err := buildWindows(cfg, loc)
	if err != nil {
		t.Fatal(err)
	}
	if w[0].Start.Location().String() != "Europe/Madrid" {
		t.Fatalf("unexpected location: %s", w[0].Start.Location())
	}
}

func TestBuildPolicy(t *testing.T) {
	cfg := config.Config{Polling: config.PollingConfig{RoundIntervals: map[string]time.Duration{"1": time.Minute, "bad": time.Second}}}
	p := buildPolicy(cfg)
	if len(p.RoundIntervals) != 1 || p.RoundIntervals[1] != time.Minute {
		t.Fatal("expected valid integer keys only")
	}
}
