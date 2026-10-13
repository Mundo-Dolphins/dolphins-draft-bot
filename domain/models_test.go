package domain

import "testing"

func TestMiamiRulesTradeLogic(t *testing.T) {
	r := NewMiamiRules("15", "MIA", "Miami Dolphins", []int{11, 30})
	out := DraftPick{Overall: 30, CurrentTeam: Team{ID: "10", Abbr: "DEN"}}
	in := DraftPick{Overall: 52, CurrentTeam: Team{ID: "15", Abbr: "MIA"}}
	if !r.IsTradeOutFromMiami(out) {
		t.Fatal("expected trade out")
	}
	if !r.IsTradeInForMiami(in) {
		t.Fatal("expected trade in")
	}
}

func TestCompletedSelection(t *testing.T) {
	p := DraftPick{PlayerName: "X", SelectionFilled: true}
	if !p.IsCompletedSelection() {
		t.Fatal("expected completed")
	}
}
