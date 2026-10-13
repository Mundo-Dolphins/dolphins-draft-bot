package espn

import (
	"os"
	"path/filepath"
	"testing"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "test", "fixtures", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParserDraftPayload(t *testing.T) {
	p := NewParser()
	snap, kind, err := p.Parse(loadFixture(t, "draft_payload.json"))
	if err != nil {
		t.Fatal(err)
	}
	if kind != kindDraft {
		t.Fatalf("expected draft kind, got %s", kind)
	}
	if len(snap.Picks) != 3 {
		t.Fatalf("unexpected picks: %d", len(snap.Picks))
	}
}

func TestParserHeaderPayload(t *testing.T) {
	p := NewParser()
	snap, kind, err := p.Parse(loadFixture(t, "header_payload.json"))
	if err != nil {
		t.Fatal(err)
	}
	if kind != kindHeader {
		t.Fatalf("expected header kind, got %s", kind)
	}
	if snap.Picks[0].PlayerName == "" {
		t.Fatal("expected player name from nested structure")
	}
}

func TestParserIncompletePayload(t *testing.T) {
	p := NewParser()
	snap, _, err := p.Parse(loadFixture(t, "incomplete_payload.json"))
	if err != nil {
		t.Fatal(err)
	}
	if snap.Picks[0].CurrentTeam.Abbr != "MIA" {
		t.Fatal("expected parsed optional fields")
	}
}

func TestParserUnexpectedShape(t *testing.T) {
	p := NewParser()
	_, _, err := p.Parse(loadFixture(t, "unexpected_shape.json"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParserInvalidJSON(t *testing.T) {
	p := NewParser()
	_, _, err := p.Parse(loadFixture(t, "invalid.json"))
	if err == nil {
		t.Fatal("expected error")
	}
}
