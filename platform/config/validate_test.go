package config

import "testing"

func baseConfig() Config {
	return Config{
		App:      AppConfig{Timezone: "Europe/Madrid"},
		Draft:    DraftConfig{ESPNURL: "http://example.com", MiamiTeamAbbr: "MIA", OriginalMiamiPicks: []int{11}},
		Windows:  []WindowConfig{{Name: "r1", Start: "2026-04-24T02:00:00", Rounds: []int{1}}},
		HTTP:     HTTPConfig{Timeout: 1},
		Telegram: TelegramConfig{BotToken: "x", ChatID: "y"},
	}
}

func TestValidateBranches(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*Config)
	}{
		{"bad timezone", func(c *Config) { c.App.Timezone = "bad" }},
		{"missing espn", func(c *Config) { c.Draft.ESPNURL = "" }},
		{"missing miami", func(c *Config) { c.Draft.MiamiTeamAbbr, c.Draft.MiamiTeamID = "", "" }},
		{"missing picks", func(c *Config) { c.Draft.OriginalMiamiPicks = nil }},
		{"missing windows", func(c *Config) { c.Windows = nil }},
		{"bad window", func(c *Config) { c.Windows = []WindowConfig{{}} }},
		{"bad timeout", func(c *Config) { c.HTTP.Timeout = 0 }},
		{"missing token", func(c *Config) { c.Telegram.BotToken = "" }},
		{"missing chat", func(c *Config) { c.Telegram.ChatID = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := baseConfig()
			tc.mut(&cfg)
			if err := Validate(cfg); err == nil {
				t.Fatal("expected error")
			}
		})
	}
	if err := Validate(baseConfig()); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestValidateWithOptionsAllowsOfflineTools(t *testing.T) {
	cfg := baseConfig()
	cfg.Telegram.BotToken = ""
	cfg.Telegram.ChatID = ""
	if err := ValidateWithOptions(cfg, LoadOptions{RequireTelegram: false}); err != nil {
		t.Fatalf("expected offline validation to pass, got %v", err)
	}
}
