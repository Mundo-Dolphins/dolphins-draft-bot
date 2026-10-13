package application

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/domain"
)

type fakeESPN struct {
	snap domain.DraftSnapshot
	err  error
}

func (f fakeESPN) FetchDraft(context.Context) (domain.DraftSnapshot, error) {
	return f.snap, f.err
}

type fakePublisher struct {
	messages []string
	err      error
	mu       sync.Mutex
}

func (f *fakePublisher) Publish(_ context.Context, message string) error {
	if f.err != nil {
		return f.err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages = append(f.messages, message)
	return nil
}

type memRepo struct {
	keys map[domain.EventKey]domain.PublishedEvent
}

func (m *memRepo) Has(_ context.Context, key domain.EventKey) (bool, error) {
	_, ok := m.keys[key]
	return ok, nil
}
func (m *memRepo) Save(_ context.Context, event domain.PublishedEvent) error {
	m.keys[event.Key] = event
	return nil
}
func (m *memRepo) List(context.Context) ([]domain.PublishedEvent, error) {
	out := make([]domain.PublishedEvent, 0, len(m.keys))
	for _, v := range m.keys {
		out = append(out, v)
	}
	return out, nil
}

func TestDraftServicePublishesSelectionAndTradesWithoutDuplicates(t *testing.T) {
	repo := &memRepo{keys: map[domain.EventKey]domain.PublishedEvent{}}
	pub := &fakePublisher{}
	rules := domain.NewMiamiRules("15", "MIA", "Miami Dolphins", []int{11, 30})
	s := NewDraftService(fakeESPN{snap: domain.DraftSnapshot{Status: domain.DraftStatus{CurrentRound: 1, DraftActive: true}, Picks: []domain.DraftPick{
		{Overall: 11, Round: 1, CurrentTeam: domain.Team{ID: "15", DisplayName: "Miami Dolphins"}, PlayerName: "A", Position: "QB", College: "X", SelectionFilled: true},
		{Overall: 30, Round: 1, CurrentTeam: domain.Team{ID: "10", DisplayName: "Denver Broncos"}},
		{Overall: 52, Round: 2, CurrentTeam: domain.Team{ID: "15", DisplayName: "Miami Dolphins"}, OriginalTeam: &domain.Team{DisplayName: "Green Bay Packers"}},
	}}}, pub, repo, rules, slog.Default(), false)
	s.now = func() time.Time { return time.Unix(0, 0) }
	if _, err := s.PollOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(pub.messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(pub.messages))
	}
	if _, err := s.PollOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(pub.messages) != 3 {
		t.Fatalf("expected no duplicates, got %d", len(pub.messages))
	}
}

func TestDraftServiceDoesNotSaveWhenPublishFails(t *testing.T) {
	repo := &memRepo{keys: map[domain.EventKey]domain.PublishedEvent{}}
	pub := &fakePublisher{err: errors.New("telegram fail")}
	rules := domain.NewMiamiRules("15", "MIA", "Miami Dolphins", []int{11})
	s := NewDraftService(fakeESPN{snap: domain.DraftSnapshot{Status: domain.DraftStatus{CurrentRound: 1, DraftActive: true}, Picks: []domain.DraftPick{{Overall: 11, CurrentTeam: domain.Team{ID: "15"}, PlayerName: "A", SelectionFilled: true}}}}, pub, repo, rules, slog.Default(), false)
	if _, err := s.PollOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(repo.keys) != 0 {
		t.Fatal("should not persist if publish fails")
	}
}
