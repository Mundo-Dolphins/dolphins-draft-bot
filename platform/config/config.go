package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	App      AppConfig      `yaml:"app"`
	Logging  LoggingConfig  `yaml:"logging"`
	Draft    DraftConfig    `yaml:"draft"`
	Windows  []WindowConfig `yaml:"windows"`
	Polling  PollingConfig  `yaml:"polling"`
	HTTP     HTTPConfig     `yaml:"http"`
	Storage  StorageConfig  `yaml:"storage"`
	Telegram TelegramConfig `yaml:"-"`
}

type AppConfig struct {
	Env      string `yaml:"env"`
	Timezone string `yaml:"timezone"`
	DryRun   bool   `yaml:"dry_run"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type DraftConfig struct {
	Year               int    `yaml:"year"`
	ESPNURL            string `yaml:"espn_url"`
	MiamiTeamID        string `yaml:"miami_team_id"`
	MiamiTeamAbbr      string `yaml:"miami_team_abbr"`
	MiamiDisplayName   string `yaml:"miami_display_name"`
	OriginalMiamiPicks []int  `yaml:"original_miami_picks"`
}

type WindowConfig struct {
	Name   string `yaml:"name"`
	Start  string `yaml:"start"`
	Rounds []int  `yaml:"rounds"`
}

type PollingConfig struct {
	RoundIntervals map[string]time.Duration `yaml:"round_intervals"`
}

type HTTPConfig struct {
	Timeout time.Duration `yaml:"timeout"`
	Retries int           `yaml:"retries"`
	Backoff time.Duration `yaml:"backoff"`
}

type StorageConfig struct {
	Kind string `yaml:"kind"`
	Path string `yaml:"path"`
}

type TelegramConfig struct {
	BotToken string
	ChatID   string
}

type LoadOptions struct {
	RequireTelegram bool
}

func Load(path string, envPath string) (Config, error) {
	return LoadWithOptions(path, envPath, LoadOptions{RequireTelegram: true})
}

func LoadWithOptions(path string, envPath string, opts LoadOptions) (Config, error) {
	var cfg Config
	raw, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read yaml config: %w", err)
	}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("parse yaml config: %w", err)
	}
	if envPath != "" {
		_ = godotenv.Load(envPath)
	} else {
		_ = godotenv.Load()
	}
	cfg.Telegram.BotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	cfg.Telegram.ChatID = os.Getenv("TELEGRAM_CHAT_ID")
	if err := ValidateWithOptions(cfg, opts); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func Validate(cfg Config) error {
	return ValidateWithOptions(cfg, LoadOptions{RequireTelegram: true})
}

func ValidateWithOptions(cfg Config, opts LoadOptions) error {
	if cfg.App.Timezone == "" {
		return errors.New("config validation: app.timezone is required")
	}
	if _, err := time.LoadLocation(cfg.App.Timezone); err != nil {
		return fmt.Errorf("config validation: invalid timezone: %w", err)
	}
	if cfg.Draft.ESPNURL == "" {
		return errors.New("config validation: draft.espn_url is required")
	}
	if cfg.Draft.MiamiTeamAbbr == "" && cfg.Draft.MiamiTeamID == "" {
		return errors.New("config validation: draft.miami_team_id or draft.miami_team_abbr is required")
	}
	if len(cfg.Draft.OriginalMiamiPicks) == 0 {
		return errors.New("config validation: draft.original_miami_picks is required")
	}
	if len(cfg.Windows) == 0 {
		return errors.New("config validation: windows is required")
	}
	for _, w := range cfg.Windows {
		if w.Name == "" || w.Start == "" || len(w.Rounds) == 0 {
			return fmt.Errorf("config validation: invalid window: %+v", w)
		}
	}
	if cfg.HTTP.Timeout <= 0 {
		return errors.New("config validation: http.timeout must be > 0")
	}
	if opts.RequireTelegram {
		if cfg.Telegram.BotToken == "" {
			return errors.New("config validation: TELEGRAM_BOT_TOKEN is required in .env")
		}
		if cfg.Telegram.ChatID == "" {
			return errors.New("config validation: TELEGRAM_CHAT_ID is required in .env")
		}
	}
	return nil
}
