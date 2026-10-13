package runner

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/platform/config"
)

func TestRunWithNoWindowsReturns(t *testing.T) {
	cfg := config.Config{
		App:      config.AppConfig{Timezone: "Europe/Madrid", DryRun: true},
		Draft:    config.DraftConfig{Year: 2026, ESPNURL: "http://127.0.0.1:1", MiamiTeamID: "15", MiamiTeamAbbr: "MIA", MiamiDisplayName: "Miami Dolphins", OriginalMiamiPicks: []int{11}},
		Polling:  config.PollingConfig{RoundIntervals: map[string]time.Duration{"1": time.Second}},
		HTTP:     config.HTTPConfig{Timeout: time.Millisecond, Retries: 0, Backoff: time.Millisecond},
		Storage:  config.StorageConfig{Kind: "json", Path: t.TempDir() + "/state.json"},
		Telegram: config.TelegramConfig{BotToken: "x", ChatID: "y"},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := Run(cfg, logger); err != nil {
		t.Fatal(err)
	}
}
