package runner

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/adapters/outbound/espn"
	"github.com/Mundo-Dolphins/dolphins-draft-bot/adapters/outbound/storage"
	"github.com/Mundo-Dolphins/dolphins-draft-bot/adapters/outbound/telegram"
	"github.com/Mundo-Dolphins/dolphins-draft-bot/application"
	"github.com/Mundo-Dolphins/dolphins-draft-bot/domain"
	"github.com/Mundo-Dolphins/dolphins-draft-bot/platform/config"
)

func Run(cfg config.Config, logger *slog.Logger) error {
	location, _ := time.LoadLocation(cfg.App.Timezone)
	windows, err := buildWindows(cfg, location)
	if err != nil {
		return err
	}
	logger.Debug("draft windows built", "count", len(windows), "timezone", cfg.App.Timezone)
	policy := buildPolicy(cfg)
	logger.Debug("polling policy built", "configured_rounds", len(policy.RoundIntervals))
	rules := domain.NewMiamiRules(cfg.Draft.MiamiTeamID, cfg.Draft.MiamiTeamAbbr, cfg.Draft.MiamiDisplayName, cfg.Draft.OriginalMiamiPicks)
	httpClient := &http.Client{Timeout: cfg.HTTP.Timeout}
	espnClient := espn.NewClient(cfg.Draft.ESPNURL, httpClient, cfg.HTTP.Retries, cfg.HTTP.Backoff, logger)
	publisher := telegram.NewPublisher(cfg.Telegram.BotToken, cfg.Telegram.ChatID, httpClient, cfg.HTTP.Retries, cfg.HTTP.Backoff)
	repo := storage.NewJSONEventRepository(cfg.Storage.Path)
	service := application.NewDraftService(espnClient, publisher, repo, rules, logger, cfg.App.DryRun)
	scheduler := application.NewScheduler(RealClock{}, service, windows, policy, logger)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	logger.Info("starting bot", "draft_year", cfg.Draft.Year, "dry_run", cfg.App.DryRun, "espn_url", cfg.Draft.ESPNURL)
	return scheduler.Run(ctx)
}

func buildWindows(cfg config.Config, loc *time.Location) ([]domain.DraftDayWindow, error) {
	windows := make([]domain.DraftDayWindow, 0, len(cfg.Windows))
	for _, w := range cfg.Windows {
		t, err := time.ParseInLocation("2006-01-02T15:04:05", w.Start, loc)
		if err != nil {
			return nil, fmt.Errorf("invalid window start %s: %w", w.Name, err)
		}
		windows = append(windows, domain.DraftDayWindow{Name: w.Name, Start: t, Rounds: w.Rounds})
	}
	return windows, nil
}

func buildPolicy(cfg config.Config) domain.PollingPolicy {
	m := make(map[int]time.Duration, len(cfg.Polling.RoundIntervals))
	for k, v := range cfg.Polling.RoundIntervals {
		parsed, err := strconv.Atoi(k)
		if err != nil {
			continue
		}
		m[parsed] = v
	}
	return domain.PollingPolicy{RoundIntervals: m}
}
