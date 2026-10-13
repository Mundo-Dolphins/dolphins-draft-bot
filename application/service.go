package application

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/domain"
	"github.com/Mundo-Dolphins/dolphins-draft-bot/ports"
)

type DraftService struct {
	espn                      ports.ESPNReader
	publisher                 ports.TelegramPublisher
	repo                      ports.EventRepository
	rules                     domain.MiamiRules
	formatter                 MessageFormatter
	logger                    *slog.Logger
	now                       func() time.Time
	dryRun                    bool
	bootstrappedOriginalPicks bool
}

func NewDraftService(espn ports.ESPNReader, pub ports.TelegramPublisher, repo ports.EventRepository, rules domain.MiamiRules, logger *slog.Logger, dryRun bool) *DraftService {
	return &DraftService{espn: espn, publisher: pub, repo: repo, rules: rules, formatter: MessageFormatter{TeamName: rules.TeamName}, logger: logger, now: time.Now, dryRun: dryRun}
}

func (s *DraftService) BootstrapOriginalPicks(ctx context.Context) error {
	if s.bootstrappedOriginalPicks {
		return nil
	}
	s.logger.Info("bootstrapping original miami picks from espn before execution window")
	snap, err := s.espn.FetchDraft(ctx)
	if err != nil {
		return err
	}
	if len(snap.Picks) == 0 {
		return fmt.Errorf("espn snapshot returned no picks for bootstrap")
	}
	picks := currentMiamiPickOveralls(snap.Picks, s.rules)
	if len(picks) == 0 {
		return fmt.Errorf("could not infer miami original picks from espn snapshot")
	}
	prevCount := len(s.rules.OriginalPicks)
	s.rules = domain.NewMiamiRules(s.rules.TeamID, s.rules.TeamAbbr, s.rules.TeamName, picks)
	s.bootstrappedOriginalPicks = true
	s.logger.Info("miami original picks refreshed", "previous_count", prevCount, "count", len(picks), "picks", picks)
	return nil
}

func (s *DraftService) PollOnce(ctx context.Context) (domain.DraftStatus, error) {
	s.logger.Debug("fetching draft snapshot from espn")
	snap, err := s.espn.FetchDraft(ctx)
	if err != nil {
		return domain.DraftStatus{}, err
	}
	s.logger.Info("draft snapshot loaded", "picks", len(snap.Picks), "current_round", snap.Status.CurrentRound)
	for _, p := range snap.Picks {
		if p.Overall <= 0 {
			continue
		}
		s.logger.Debug("processing pick", "overall", p.Overall, "round", p.Round, "selection_filled", p.SelectionFilled)
		if err := s.processPick(ctx, p); err != nil {
			s.logger.Error("failed to process pick", "overall", p.Overall, "error", err)
		}
	}
	return snap.Status, nil
}

func (s *DraftService) processPick(ctx context.Context, p domain.DraftPick) error {
	s.logger.Debug("evaluating pick event rules", "overall", p.Overall)
	if p.IsCompletedSelection() && s.rules.BelongsToMiamiNow(p) {
		s.logger.Debug("pick matched selection event", "overall", p.Overall)
		return s.publishIfNew(ctx, domain.EventSelection, p.Overall, s.formatter.Selection(p))
	}
	if s.rules.IsTradeOutFromMiami(p) {
		s.logger.Debug("pick matched trade-out event", "overall", p.Overall)
		return s.publishIfNew(ctx, domain.EventTradeOut, p.Overall, s.formatter.TradeOut(p.Overall, p.CurrentTeam.DisplayName))
	}
	if s.rules.IsTradeInForMiami(p) {
		s.logger.Debug("pick matched trade-in event", "overall", p.Overall)
		source := ""
		if p.OriginalTeam != nil {
			source = p.OriginalTeam.DisplayName
		}
		return s.publishIfNew(ctx, domain.EventTradeIn, p.Overall, s.formatter.TradeIn(p.Overall, source))
	}
	s.logger.Debug("pick produced no publishable event", "overall", p.Overall)
	return nil
}

func (s *DraftService) publishIfNew(ctx context.Context, t domain.EventType, overall int, msg string) error {
	key := domain.BuildEventKey(t, overall)
	s.logger.Debug("checking event deduplication", "event_key", key)
	exists, err := s.repo.Has(ctx, key)
	if err != nil {
		return err
	}
	if exists {
		s.logger.Info("event already published", "event_key", key)
		return nil
	}
	if s.dryRun {
		s.logger.Info("dry run message", "event_key", key, "message", msg)
	} else {
		s.logger.Debug("publishing telegram message", "event_key", key, "message", msg)
		if err := s.publisher.Publish(ctx, msg); err != nil {
			return err
		}
		s.logger.Info("telegram message published", "event_key", key)
	}
	if err := s.repo.Save(ctx, domain.PublishedEvent{Key: key, Type: t, Overall: overall, CreatedAt: s.now()}); err != nil {
		return err
	}
	s.logger.Debug("event persisted", "event_key", key)
	return nil
}

func currentMiamiPickOveralls(picks []domain.DraftPick, rules domain.MiamiRules) []int {
	set := make(map[int]struct{})
	originals := make([]int, 0)
	for _, p := range picks {
		if p.Overall <= 0 {
			continue
		}
		if !rules.BelongsToMiamiNow(p) {
			continue
		}
		if _, exists := set[p.Overall]; exists {
			continue
		}
		set[p.Overall] = struct{}{}
		originals = append(originals, p.Overall)
	}
	sort.Ints(originals)
	return originals
}
