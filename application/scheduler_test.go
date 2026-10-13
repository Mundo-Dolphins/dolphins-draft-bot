package application

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/domain"
)

type fakeClock struct {
	now    time.Time
	sleeps []time.Duration
	limit  int
}

func (f *fakeClock) Now() time.Time { return f.now }
func (f *fakeClock) Sleep(_ context.Context, d time.Duration) error {
	f.sleeps = append(f.sleeps, d)
	f.now = f.now.Add(d)
	if len(f.sleeps) >= f.limit {
		return context.Canceled
	}
	return nil
}

type fakeService struct {
	statuses []domain.DraftStatus
	idx      int
	err      error
}

func (f *fakeService) PollOnce(context.Context) (domain.DraftStatus, error) {
	if f.err != nil {
		return domain.DraftStatus{}, f.err
	}
	if f.idx >= len(f.statuses) {
		return domain.DraftStatus{CurrentRound: 7, DraftActive: false}, nil
	}
	s := f.statuses[f.idx]
	f.idx++
	return s, nil
}

func TestSchedulerWaitsForWindowAndUsesRoundPolicy(t *testing.T) {
	clk := &fakeClock{now: time.Date(2026, 4, 24, 1, 0, 0, 0, time.FixedZone("MAD", 3600)), limit: 2}
	service := &fakeService{statuses: []domain.DraftStatus{{CurrentRound: 1, DraftActive: true}}}
	s := &Scheduler{clock: clk, service: service, windows: []domain.DraftDayWindow{{Name: "r1", Start: time.Date(2026, 4, 24, 2, 0, 0, 0, time.FixedZone("MAD", 3600)), Rounds: []int{1}}}, policy: domain.PollingPolicy{RoundIntervals: map[int]time.Duration{1: time.Minute}}, logger: slog.Default()}
	_ = s.Run(context.Background())
	if len(clk.sleeps) == 0 || clk.sleeps[0] != time.Hour {
		t.Fatalf("expected first sleep of 1h, got %#v", clk.sleeps)
	}
}

func TestWindowFinished(t *testing.T) {
	s := Scheduler{}
	w := domain.DraftDayWindow{Rounds: []int{2, 3}}
	if !s.windowFinished(domain.DraftStatus{DraftActive: false}, w) {
		t.Fatal("inactive draft should finish window")
	}
	if !s.windowFinished(domain.DraftStatus{DraftActive: true, CurrentRound: 4}, w) {
		t.Fatal("round out of window should finish window")
	}
	if s.windowFinished(domain.DraftStatus{DraftActive: true, CurrentRound: 2}, w) {
		t.Fatal("round in window should not finish")
	}
}

func TestSchedulerPollErrorSleepsRetry(t *testing.T) {
	clk := &fakeClock{now: time.Now(), limit: 1}
	service := &fakeService{err: errors.New("boom")}
	s := &Scheduler{clock: clk, service: service, windows: []domain.DraftDayWindow{{Name: "x", Start: time.Now().Add(-time.Minute), Rounds: []int{1}}}, policy: domain.PollingPolicy{RoundIntervals: map[int]time.Duration{1: time.Second}}, logger: slog.Default()}
	_ = s.Run(context.Background())
	if len(clk.sleeps) != 1 || clk.sleeps[0] != 10*time.Second {
		t.Fatalf("unexpected sleeps: %#v", clk.sleeps)
	}
}
