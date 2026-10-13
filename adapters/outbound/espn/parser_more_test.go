package espn

import "testing"

func TestDetectorAndHelpers(t *testing.T) {
	d := Detector{}
	if _, err := d.Detect([]byte("{")); err == nil {
		t.Fatal("expected invalid json")
	}
	if _, err := d.Detect([]byte(`{"draft":{}}`)); err != nil {
		t.Fatal(err)
	}
	if toString(12.0) != "12" || toString(7) != "7" || toString("x") != "x" || toString(nil) != "" {
		t.Fatal("toString unexpected")
	}
	e := StructuredParseError{Kind: "k", Message: "m"}
	if e.Error() != "k: m" {
		t.Fatal("bad error string")
	}
	p := NewParser()
	if _, err := p.parseByKind([]byte(`{}`), kindUnknown); err == nil {
		t.Fatal("expected unsupported kind error")
	}
}

func TestParserDraftPayload_NewESPNSchema(t *testing.T) {
	raw := []byte(`{
		"status": {"round": 1, "state": "in", "name": "BETWEEN_ROUNDS"},
		"teams": [{"id": "15", "abbreviation": "MIA", "displayName": "Miami Dolphins"}],
		"picks": [
			{
				"overall": 11,
				"round": 1,
				"pick": 11,
				"status": "SELECTION_MADE",
				"teamId": "15",
				"athlete": {
					"displayName": "Caleb Downs",
					"position": {"abbreviation": "S"},
					"team": {"shortDisplayName": "Ohio State"}
				}
			},
			{
				"overall": 33,
				"round": 2,
				"pick": 1,
				"status": "ON_THE_CLOCK",
				"teamId": "25"
			}
		]
	}`)

	p := NewParser()
	snap, kind, err := p.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if kind != kindDraft {
		t.Fatalf("expected draft kind, got %s", kind)
	}
	if snap.Status.CurrentRound != 1 {
		t.Fatalf("expected round 1, got %d", snap.Status.CurrentRound)
	}
	if len(snap.Picks) != 2 {
		t.Fatalf("expected 2 picks, got %d", len(snap.Picks))
	}
	if snap.Picks[0].PlayerName != "Caleb Downs" {
		t.Fatalf("expected player name from athlete, got %q", snap.Picks[0].PlayerName)
	}
	if !snap.Picks[0].SelectionFilled {
		t.Fatal("expected SELECTION_MADE to be treated as selection filled")
	}
	if snap.Picks[0].CurrentTeam.Abbr != "MIA" {
		t.Fatalf("expected team abbreviation from teams list, got %q", snap.Picks[0].CurrentTeam.Abbr)
	}
	if snap.Picks[1].SelectionFilled {
		t.Fatal("expected ON_THE_CLOCK pick to be not selection filled")
	}
}

func TestParseDraftPayload_PickAsNameOrNumber(t *testing.T) {
	raw := []byte(`{"picks":[
		{"overall":1,"round":1,"pick":"Player One"},
		{"overall":2,"round":1,"pick":2},
		{"overall":3,"round":1,"pick":null}
	]}`)
	snap, err := parseDraftPayload(raw)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Player One", "", ""}
	for i, w := range want {
		if got := snap.Picks[i].PlayerName; got != w {
			t.Fatalf("pick %d: expected player %q, got %q", i, w, got)
		}
	}
}
