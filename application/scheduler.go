package application

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/domain"
	"github.com/Mundo-Dolphins/dolphins-draft-bot/ports"
)

type Scheduler struct {
	clock        ports.Clock
	service      poller
	bootstrapper preWindowBootstrapper
	windows      []domain.DraftDayWindow
	policy       domain.PollingPolicy
	logger       *slog.Logger
}

type poller interface {
	PollOnce(ctx context.Context) (domain.DraftStatus, error)
}

type preWindowBootstrapper interface {
	BootstrapOriginalPicks(ctx context.Context) error
}

func NewScheduler(clock ports.Clock, service poller, windows []domain.DraftDayWindow, policy domain.PollingPolicy, logger *slog.Logger) *Scheduler {
	domain.SortWindows(windows)
	s := &Scheduler{clock: clock, service: service, windows: windows, policy: policy, logger: logger}
	if b, ok := service.(preWindowBootstrapper); ok {
		s.bootstrapper = b
	}
	return s
}

func (s *Scheduler) Run(ctx context.Context) error {
	activeWindow := -1
	for {
		now := s.clock.Now()
		s.logger.Debug("scheduler tick", "now", now.Format(time.RFC3339))
		idx := s.activeWindowIndex(now)
		if idx < 0 {
			if activeWindow >= 0 {
				s.logger.Info("execution window finished", "window", s.windows[activeWindow].Name)
				activeWindow = -1
			}
			next, ok := s.nextWindow(now)
			if !ok {
				s.logger.Info("no more draft windows")
				return nil
			}
			wait := next.Start.Sub(now)
			s.logger.Info("sleeping until next window", "window", next.Name, "starts_at", next.Start.Format(time.RFC3339), "duration", wait.String())
			if err := s.clock.Sleep(ctx, wait); err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				return err
			}
			continue
		}
		w := s.windows[idx]
		if activeWindow != idx {
			if s.bootstrapper != nil {
				s.logger.Debug("running pre-window bootstrap", "window", w.Name)
				if err := s.bootstrapper.BootstrapOriginalPicks(ctx); err != nil {
					s.logger.Error("pre-window bootstrap failed", "window", w.Name, "error", err)
					s.logger.Debug("bootstrap retry sleep", "duration", (10 * time.Second).String())
					if err := s.clock.Sleep(ctx, 10*time.Second); err != nil {
						return nil
					}
					continue
				}
			}
			s.logger.Info("execution window started", "window", w.Name, "rounds", w.Rounds, "start", w.Start.Format(time.RFC3339))
			activeWindow = idx
		}
		s.logger.Debug("polling draft status", "window", w.Name)
		status, err := s.service.PollOnce(ctx)
		if err != nil {
			s.logger.Error("poll failed", "error", err)
			s.logger.Debug("poll retry sleep", "duration", (10 * time.Second).String())
			if err := s.clock.Sleep(ctx, 10*time.Second); err != nil {
				return nil
			}
			continue
		}
		s.logger.Debug("draft status polled", "draft_active", status.DraftActive, "current_round", status.CurrentRound)
		if s.windowFinished(status, w) {
			s.logger.Info("window no longer active for current status", "window", w.Name, "draft_active", status.DraftActive, "current_round", status.CurrentRound)
			activeWindow = -1
			next, ok := s.nextWindow(now)
			if !ok {
				return nil
			}
			s.logger.Info("sleeping until next window", "window", next.Name, "starts_at", next.Start.Format(time.RFC3339), "duration", next.Start.Sub(now).String())
			if err := s.clock.Sleep(ctx, next.Start.Sub(now)); err != nil {
				return nil
			}
			continue
		}
		interval := s.policy.IntervalForRound(status.CurrentRound)
		s.logger.Info("polling sleep", "round", status.CurrentRound, "interval", interval.String())
		if err := s.clock.Sleep(ctx, interval); err != nil {
			return nil
		}
	}
}

func (s *Scheduler) activeWindowIndex(now time.Time) int {
	for i := range s.windows {
		if now.Before(s.windows[i].Start) {
			continue
		}
		if i+1 < len(s.windows) && !now.Before(s.windows[i+1].Start) {
			continue
		}
		return i
	}
	return -1
}

func (s *Scheduler) nextWindow(now time.Time) (domain.DraftDayWindow, bool) {
	for _, w := range s.windows {
		if !now.After(w.Start) {
			return w, true
		}
	}
	return domain.DraftDayWindow{}, false
}

func (s *Scheduler) windowFinished(status domain.DraftStatus, w domain.DraftDayWindow) bool {
	if !status.DraftActive {
		return true
	}
	return !domain.IsRoundInWindow(status.CurrentRound, w)
}
