package application

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/domain"
)

type fakeBootstrapService struct {
	statuses       []domain.DraftStatus
	idx            int
	bootstrapCalls int
	bootstrapErr   error
}

func (f *fakeBootstrapService) PollOnce(context.Context) (domain.DraftStatus, error) {
	if f.idx >= len(f.statuses) {
		return domain.DraftStatus{CurrentRound: 1, DraftActive: true}, nil
	}
	s := f.statuses[f.idx]
	f.idx++
	return s, nil
}

func (f *fakeBootstrapService) BootstrapOriginalPicks(context.Context) error {
	f.bootstrapCalls++
	return f.bootstrapErr
}

func TestSchedulerHelpers(t *testing.T) {
	s := NewScheduler(&fakeClock{}, &fakeService{}, []domain.DraftDayWindow{{Name: "b", Start: time.Unix(2, 0)}, {Name: "a", Start: time.Unix(1, 0)}}, domain.PollingPolicy{}, slog.Default())
	if s.activeWindowIndex(time.Unix(0, 0)) != -1 {
		t.Fatal("expected no active window")
	}
	if _, ok := s.nextWindow(time.Unix(3, 0)); ok {
		t.Fatal("expected no next window")
	}
}

func TestSchedulerWindowBoundaryUsesNewWindow(t *testing.T) {
	windows := []domain.DraftDayWindow{{Name: "first", Start: time.Unix(1, 0)}, {Name: "second", Start: time.Unix(2, 0)}}
	s := NewScheduler(&fakeClock{}, &fakeService{}, windows, domain.PollingPolicy{}, slog.Default())
	if idx := s.activeWindowIndex(time.Unix(2, 0)); idx != 1 {
		t.Fatalf("expected second window at boundary, got %d", idx)
	}
	next, ok := s.nextWindow(time.Unix(2, 0))
	if !ok || next.Name != "second" {
		t.Fatalf("expected second window as next at boundary, got ok=%v next=%+v", ok, next)
	}
}

func TestSchedulerRunsBootstrapBeforeWindowStart(t *testing.T) {
	clk := &fakeClock{now: time.Unix(2, 0), limit: 1}
	svc := &fakeBootstrapService{statuses: []domain.DraftStatus{{CurrentRound: 1, DraftActive: true}}}
	s := NewScheduler(
		clk,
		svc,
		[]domain.DraftDayWindow{{Name: "r1", Start: time.Unix(1, 0), Rounds: []int{1}}},
		domain.PollingPolicy{RoundIntervals: map[int]time.Duration{1: time.Second}},
		slog.Default(),
	)

	_ = s.Run(context.Background())
	if svc.bootstrapCalls != 1 {
		t.Fatalf("expected bootstrap once, got %d", svc.bootstrapCalls)
	}
	if svc.idx != 1 {
		t.Fatalf("expected one poll after bootstrap, got %d", svc.idx)
	}
}

func TestSchedulerRetriesWhenBootstrapFails(t *testing.T) {
	clk := &fakeClock{now: time.Unix(2, 0), limit: 1}
	svc := &fakeBootstrapService{bootstrapErr: errors.New("boom")}
	s := NewScheduler(
		clk,
		svc,
		[]domain.DraftDayWindow{{Name: "r1", Start: time.Unix(1, 0), Rounds: []int{1}}},
		domain.PollingPolicy{RoundIntervals: map[int]time.Duration{1: time.Second}},
		slog.Default(),
	)

	_ = s.Run(context.Background())
	if svc.bootstrapCalls != 1 {
		t.Fatalf("expected one bootstrap attempt, got %d", svc.bootstrapCalls)
	}
	if svc.idx != 0 {
		t.Fatal("poll should not run when bootstrap fails")
	}
	if len(clk.sleeps) != 1 || clk.sleeps[0] != 10*time.Second {
		t.Fatalf("expected bootstrap retry sleep of 10s, got %#v", clk.sleeps)
	}
}
