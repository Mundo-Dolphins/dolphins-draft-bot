package domain

import (
	"sort"
	"testing"
	"time"
)

func TestPolicyAndHelpers(t *testing.T) {
	p := PollingPolicy{RoundIntervals: map[int]time.Duration{7: 25 * time.Second, 1: time.Minute}}
	if p.IntervalForRound(1) != time.Minute || p.IntervalForRound(8) != 25*time.Second {
		t.Fatal("interval fallback failed")
	}
	if BuildEventKey(EventTradeIn, 52) != "trade_in:52" {
		t.Fatal("bad event key")
	}
	w := []DraftDayWindow{{Name: "b", Start: time.Unix(2, 0)}, {Name: "a", Start: time.Unix(1, 0)}}
	SortWindows(w)
	if !sort.SliceIsSorted(w, func(i, j int) bool { return w[i].Start.Before(w[j].Start) }) {
		t.Fatal("windows not sorted")
	}
	if !IsRoundInWindow(2, DraftDayWindow{Rounds: []int{1, 2}}) || IsRoundInWindow(3, DraftDayWindow{Rounds: []int{1, 2}}) {
		t.Fatal("round check failed")
	}
}
