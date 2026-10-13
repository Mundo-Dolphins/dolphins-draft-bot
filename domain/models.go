package domain

import (
	"fmt"
	"sort"
	"time"
)

type EventKey string

type EventType string

const (
	EventSelection EventType = "selection"
	EventTradeOut  EventType = "trade_out"
	EventTradeIn   EventType = "trade_in"
)

type Team struct {
	ID          string
	Abbr        string
	DisplayName string
}

type DraftPick struct {
	Overall         int
	Round           int
	CurrentTeam     Team
	OriginalTeam    *Team
	PlayerName      string
	Position        string
	College         string
	SelectionFilled bool
}

type DraftStatus struct {
	State        string
	CurrentRound int
	DraftActive  bool
}

type DraftSnapshot struct {
	Status DraftStatus
	Picks  []DraftPick
}

type PublishedEvent struct {
	Key       EventKey  `json:"key"`
	Type      EventType `json:"type"`
	Overall   int       `json:"overall"`
	CreatedAt time.Time `json:"created_at"`
}

type DraftDayWindow struct {
	Name   string
	Start  time.Time
	Rounds []int
}

type PollingPolicy struct {
	RoundIntervals map[int]time.Duration
}

func (p PollingPolicy) IntervalForRound(round int) time.Duration {
	if val, ok := p.RoundIntervals[round]; ok {
		return val
	}
	if val, ok := p.RoundIntervals[7]; ok {
		return val
	}
	return 30 * time.Second
}

type MiamiRules struct {
	TeamID        string
	TeamAbbr      string
	TeamName      string
	OriginalPicks map[int]struct{}
}

func NewMiamiRules(teamID, teamAbbr, teamName string, originals []int) MiamiRules {
	m := make(map[int]struct{}, len(originals))
	for _, p := range originals {
		m[p] = struct{}{}
	}
	return MiamiRules{TeamID: teamID, TeamAbbr: teamAbbr, TeamName: teamName, OriginalPicks: m}
}

func (r MiamiRules) BelongsToMiamiNow(p DraftPick) bool {
	if r.TeamID != "" && p.CurrentTeam.ID == r.TeamID {
		return true
	}
	return p.CurrentTeam.Abbr == r.TeamAbbr || p.CurrentTeam.DisplayName == r.TeamName
}

func (r MiamiRules) WasOriginallyMiamiPick(overall int) bool {
	_, ok := r.OriginalPicks[overall]
	return ok
}

func (r MiamiRules) IsTradeOutFromMiami(p DraftPick) bool {
	return r.WasOriginallyMiamiPick(p.Overall) && !r.BelongsToMiamiNow(p)
}

func (r MiamiRules) IsTradeInForMiami(p DraftPick) bool {
	return !r.WasOriginallyMiamiPick(p.Overall) && r.BelongsToMiamiNow(p)
}

func (p DraftPick) IsCompletedSelection() bool {
	return p.SelectionFilled && p.PlayerName != ""
}

func BuildEventKey(t EventType, overall int) EventKey {
	return EventKey(fmt.Sprintf("%s:%d", t, overall))
}

func SortWindows(w []DraftDayWindow) {
	sort.Slice(w, func(i, j int) bool { return w[i].Start.Before(w[j].Start) })
}

func IsRoundInWindow(round int, w DraftDayWindow) bool {
	for _, r := range w.Rounds {
		if r == round {
			return true
		}
	}
	return false
}
