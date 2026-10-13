package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigYAMLAndEnv(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "config.yaml")
	envPath := filepath.Join(dir, ".env")
	yaml := `app:
  env: test
  timezone: Europe/Madrid
  dry_run: true
logging:
  level: info
  format: json
draft:
  year: 2026
  espn_url: http://example.com
  miami_team_id: "15"
  miami_team_abbr: MIA
  miami_display_name: Miami Dolphins
  original_miami_picks: [11]
windows:
  - name: r1
    start: "2026-04-24T02:00:00"
    rounds: [1]
polling:
  round_intervals:
    "1": 60s
http:
  timeout: 10s
  retries: 1
  backoff: 1s
storage:
  kind: json
  path: ./state.json
`
	if err := os.WriteFile(yamlPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte("TELEGRAM_BOT_TOKEN=x\nTELEGRAM_CHAT_ID=y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(yamlPath, envPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Telegram.BotToken != "x" || cfg.Telegram.ChatID != "y" {
		t.Fatal("expected env values")
	}
}

func TestLoadMissingEnvFails(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "config.yaml")
	yaml := `app:
  env: test
  timezone: Europe/Madrid
draft:
  year: 2026
  espn_url: http://example.com
  miami_team_abbr: MIA
  original_miami_picks: [11]
windows:
  - name: r1
    start: "2026-04-24T02:00:00"
    rounds: [1]
polling:
  round_intervals:
    "1": 60s
http:
  timeout: 10s
  retries: 1
  backoff: 1s
storage:
  kind: json
  path: ./state.json
`
	if err := os.WriteFile(yamlPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(yamlPath, filepath.Join(dir, "missing.env"))
	if err == nil || !strings.Contains(err.Error(), "TELEGRAM_BOT_TOKEN") {
		t.Fatalf("expected env error, got %v", err)
	}
}

func TestInvalidYAMLFails(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte("app: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path, "")
	if err == nil {
		t.Fatal("expected yaml parsing error")
	}
}

func TestLoadWithOptionsAllowsOfflineCommands(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "config.yaml")
	yaml := `app:
  env: test
  timezone: Europe/Madrid
logging:
  level: info
  format: json
draft:
  year: 2026
  espn_url: http://example.com
  miami_team_abbr: MIA
  original_miami_picks: [11]
windows:
  - name: r1
    start: "2026-04-24T02:00:00"
    rounds: [1]
polling:
  round_intervals:
    "1": 60s
http:
  timeout: 10s
  retries: 1
  backoff: 1s
storage:
  kind: json
  path: ./state.json
`
	if err := os.WriteFile(yamlPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")
	cfg, err := LoadWithOptions(yamlPath, filepath.Join(dir, "missing.env"), LoadOptions{RequireTelegram: false})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Draft.ESPNURL != "http://example.com" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
