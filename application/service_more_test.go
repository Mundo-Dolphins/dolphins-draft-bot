package application

import (
	"context"
	"log/slog"
	"testing"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/domain"
)

type countingESPN struct {
	snap  domain.DraftSnapshot
	err   error
	calls int
}

func (c *countingESPN) FetchDraft(context.Context) (domain.DraftSnapshot, error) {
	c.calls++
	if c.err != nil {
		return domain.DraftSnapshot{}, c.err
	}
	return c.snap, nil
}

func TestDraftServiceDryRunAndSkipInvalidOverall(t *testing.T) {
	repo := &memRepo{keys: map[domain.EventKey]domain.PublishedEvent{}}
	pub := &fakePublisher{}
	rules := domain.NewMiamiRules("15", "MIA", "Miami Dolphins", []int{11})
	s := NewDraftService(fakeESPN{snap: domain.DraftSnapshot{Status: domain.DraftStatus{CurrentRound: 1, DraftActive: true}, Picks: []domain.DraftPick{{Overall: 0}, {Overall: 11, CurrentTeam: domain.Team{ID: "15"}, PlayerName: "A", SelectionFilled: true}}}}, pub, repo, rules, slog.Default(), true)
	if _, err := s.PollOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(pub.messages) != 0 {
		t.Fatal("dry run should not call publisher")
	}
	if len(repo.keys) != 1 {
		t.Fatal("dry run should persist")
	}
}

func TestDraftServiceBootstrapOriginalPicksFromESPN(t *testing.T) {
	repo := &memRepo{keys: map[domain.EventKey]domain.PublishedEvent{}}
	pub := &fakePublisher{}
	espn := &countingESPN{snap: domain.DraftSnapshot{Picks: []domain.DraftPick{
		{Overall: 12, CurrentTeam: domain.Team{ID: "15", Abbr: "MIA"}},
		{Overall: 43, CurrentTeam: domain.Team{ID: "15", Abbr: "MIA"}},
		{Overall: 75, CurrentTeam: domain.Team{ID: "10", Abbr: "DEN"}},
	}}}
	rules := domain.NewMiamiRules("15", "MIA", "Miami Dolphins", []int{11, 30})
	s := NewDraftService(espn, pub, repo, rules, slog.Default(), true)

	if err := s.BootstrapOriginalPicks(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !s.rules.WasOriginallyMiamiPick(12) || !s.rules.WasOriginallyMiamiPick(43) {
		t.Fatal("expected refreshed miami picks from espn snapshot")
	}
	if s.rules.WasOriginallyMiamiPick(11) {
		t.Fatal("expected previous configured pick list to be replaced")
	}
	if espn.calls != 1 {
		t.Fatalf("expected one espn call, got %d", espn.calls)
	}

	if err := s.BootstrapOriginalPicks(context.Background()); err != nil {
		t.Fatal(err)
	}
	if espn.calls != 1 {
		t.Fatalf("expected bootstrap to run once, got %d calls", espn.calls)
	}
}
