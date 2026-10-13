package application

import (
	"testing"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/domain"
)

func TestSelectionMessage(t *testing.T) {
	f := MessageFormatter{}
	msg := f.Selection(domain.DraftPick{Overall: 11, PlayerName: "John Doe", Position: "QB", College: "Miami"})
	expected := "The pick is in, Miami Dolphins han seleccionado con el pick 11 a John Doe, QB, Miami"
	if msg != expected {
		t.Fatalf("unexpected message: %s", msg)
	}
}

func TestTradeMessages(t *testing.T) {
	f := MessageFormatter{}
	if got := f.TradeOut(30, "Denver Broncos"); got != "Trade alert: el pick 30 de Miami Dolphins ha sido traspasado a Denver Broncos" {
		t.Fatal(got)
	}
	if got := f.TradeIn(52, "Green Bay Packers"); got != "Trade alert: Miami Dolphins ha adquirido el pick 52, originalmente de Green Bay Packers" {
		t.Fatal(got)
	}
	if got := f.TradeIn(52, ""); got != "Trade alert: Miami Dolphins ha adquirido el pick 52" {
		t.Fatal(got)
	}
}

func TestFormatterUsesConfiguredTeamName(t *testing.T) {
	f := MessageFormatter{TeamName: "Buffalo Bills"}
	if got := f.Selection(domain.DraftPick{Overall: 9, PlayerName: "John Doe", Position: "QB", College: "Miami"}); got != "The pick is in, Buffalo Bills han seleccionado con el pick 9 a John Doe, QB, Miami" {
		t.Fatal(got)
	}
	if got := f.TradeOut(30, "Denver Broncos"); got != "Trade alert: el pick 30 de Buffalo Bills ha sido traspasado a Denver Broncos" {
		t.Fatal(got)
	}
	if got := f.TradeIn(52, ""); got != "Trade alert: Buffalo Bills ha adquirido el pick 52" {
		t.Fatal(got)
	}
}
