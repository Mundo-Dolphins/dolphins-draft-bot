package espn

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/domain"
)

type payloadKind string

const (
	kindUnknown payloadKind = "unknown"
	kindDraft   payloadKind = "draft_endpoint"
	kindHeader  payloadKind = "header_endpoint"
)

type StructuredParseError struct {
	Kind    string
	Message string
}

func (e StructuredParseError) Error() string { return fmt.Sprintf("%s: %s", e.Kind, e.Message) }

type Detector struct{}

func (Detector) Detect(raw []byte) (payloadKind, error) {
	if !json.Valid(raw) {
		return kindUnknown, StructuredParseError{Kind: "invalid_json", Message: "payload is not valid JSON"}
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return kindUnknown, err
	}
	if _, ok := root["picks"]; ok {
		return kindDraft, nil
	}
	if _, ok := root["sports"]; ok {
		return kindHeader, nil
	}
	if _, ok := root["draft"]; ok {
		return kindHeader, nil
	}
	return kindUnknown, StructuredParseError{Kind: "unexpected_shape", Message: "unknown payload shape"}
}

type Parser struct {
	detector Detector
}

func NewParser() Parser {
	return Parser{detector: Detector{}}
}

func (p Parser) Parse(raw []byte) (domain.DraftSnapshot, payloadKind, error) {
	kind, err := p.detector.Detect(raw)
	if err != nil {
		if kind == kindUnknown && json.Valid(raw) {
			snap, tryErr := p.tryAll(raw)
			return snap, kindUnknown, tryErr
		}
		return domain.DraftSnapshot{}, kind, err
	}
	snap, parseErr := p.parseByKind(raw, kind)
	if parseErr == nil {
		return snap, kind, nil
	}
	fallback, fbErr := p.tryAll(raw)
	if fbErr != nil {
		return domain.DraftSnapshot{}, kind, parseErr
	}
	return fallback, kind, nil
}

func (p Parser) parseByKind(raw []byte, kind payloadKind) (domain.DraftSnapshot, error) {
	switch kind {
	case kindDraft:
		return parseDraftPayload(raw)
	case kindHeader:
		return parseHeaderPayload(raw)
	default:
		return domain.DraftSnapshot{}, StructuredParseError{Kind: "unexpected_shape", Message: "unsupported payload kind"}
	}
}

func (p Parser) tryAll(raw []byte) (domain.DraftSnapshot, error) {
	if snap, err := parseDraftPayload(raw); err == nil {
		return snap, nil
	}
	if snap, err := parseHeaderPayload(raw); err == nil {
		return snap, nil
	}
	return domain.DraftSnapshot{}, StructuredParseError{Kind: "unexpected_shape", Message: "no parser could normalize payload"}
}

type draftDTO struct {
	Status struct {
		Type struct {
			Name string `json:"name"`
		} `json:"type"`
		CurrentRound int    `json:"currentRound"`
		Round        int    `json:"round"`
		State        string `json:"state"`
		Name         string `json:"name"`
	} `json:"status"`
	Teams []struct {
		ID          interface{} `json:"id"`
		Abbr        string      `json:"abbreviation"`
		DisplayName string      `json:"displayName"`
	} `json:"teams"`
	Picks []draftPickDTO `json:"picks"`
}

type draftPickDTO struct {
	Overall     int         `json:"overall"`
	Round       int         `json:"round"`
	TeamID      interface{} `json:"teamId"`
	DisplayName string      `json:"displayName"`
	Abbr        string      `json:"shortName"`
	// ESPN uses "pick" both for the player name (string) and for the pick
	// number (integer) depending on the payload, so it is decoded lazily.
	Pick         json.RawMessage `json:"pick"`
	Status       string          `json:"status"`
	Position     string          `json:"position"`
	College      string          `json:"college"`
	SelectionVal string          `json:"selectionStatus"`
	Athlete      struct {
		DisplayName string `json:"displayName"`
		Position    struct {
			Abbr string `json:"abbreviation"`
		} `json:"position"`
		Team struct {
			Name             string `json:"name"`
			ShortDisplayName string `json:"shortDisplayName"`
		} `json:"team"`
	} `json:"athlete"`
	Team struct {
		ID          string `json:"id"`
		Abbr        string `json:"abbreviation"`
		DisplayName string `json:"displayName"`
	} `json:"team"`
	OriginalTeam struct {
		ID          string `json:"id"`
		Abbr        string `json:"abbreviation"`
		DisplayName string `json:"displayName"`
	} `json:"originalTeam"`
}

func parseDraftPayload(raw []byte) (domain.DraftSnapshot, error) {
	var dto draftDTO
	if err := json.Unmarshal(raw, &dto); err != nil {
		return domain.DraftSnapshot{}, err
	}
	if len(dto.Picks) == 0 {
		return domain.DraftSnapshot{}, StructuredParseError{Kind: "unexpected_shape", Message: "draft payload without picks"}
	}
	stateName := strings.ToLower(firstNonEmpty(dto.Status.Type.Name, dto.Status.State, dto.Status.Name))
	status := domain.DraftStatus{State: stateName, CurrentRound: firstInt(dto.Status.CurrentRound, dto.Status.Round), DraftActive: strings.Contains(stateName, "in") || strings.Contains(stateName, "active")}
	if status.CurrentRound == 0 {
		for _, p := range dto.Picks {
			if p.Round > status.CurrentRound {
				status.CurrentRound = p.Round
			}
		}
	}
	if status.State == "" {
		status.State = "unknown"
	}
	teamsByID := make(map[string]domain.Team, len(dto.Teams))
	for _, t := range dto.Teams {
		id := toString(t.ID)
		if id == "" {
			continue
		}
		teamsByID[id] = domain.Team{ID: id, Abbr: t.Abbr, DisplayName: t.DisplayName}
	}

	picks := make([]domain.DraftPick, 0, len(dto.Picks))
	for _, p := range dto.Picks {
		teamID := firstNonEmpty(p.Team.ID, toString(p.TeamID))
		team := domain.Team{ID: teamID, Abbr: firstNonEmpty(p.Team.Abbr, p.Abbr), DisplayName: firstNonEmpty(p.Team.DisplayName, p.DisplayName)}
		if mapped, ok := teamsByID[teamID]; ok {
			if team.Abbr == "" {
				team.Abbr = mapped.Abbr
			}
			if team.DisplayName == "" {
				team.DisplayName = mapped.DisplayName
			}
		}
		var original *domain.Team
		if p.OriginalTeam.DisplayName != "" || p.OriginalTeam.Abbr != "" || p.OriginalTeam.ID != "" {
			original = &domain.Team{ID: p.OriginalTeam.ID, Abbr: p.OriginalTeam.Abbr, DisplayName: p.OriginalTeam.DisplayName}
		}
		player := firstNonEmpty(p.Athlete.DisplayName, pickPlayerName(p.Pick))
		position := firstNonEmpty(p.Athlete.Position.Abbr, p.Position)
		college := firstNonEmpty(p.College, p.Athlete.Team.ShortDisplayName, p.Athlete.Team.Name)
		selectionFilled := player != "" || strings.EqualFold(p.SelectionVal, "selected") || strings.EqualFold(p.Status, "selection_made")
		picks = append(picks, domain.DraftPick{Overall: p.Overall, Round: p.Round, CurrentTeam: team, OriginalTeam: original, PlayerName: player, Position: position, College: college, SelectionFilled: selectionFilled})
	}
	if !status.DraftActive {
		status.DraftActive = status.State != "final"
	}
	return domain.DraftSnapshot{Status: status, Picks: picks}, nil
}

type headerDTO struct {
	Sports []struct {
		Leagues []struct {
			Draft struct {
				Status struct {
					CurrentRound int  `json:"currentRound"`
					IsActive     bool `json:"isActive"`
				} `json:"status"`
				Rounds []struct {
					Round int `json:"round"`
					Picks []struct {
						Overall int `json:"overall"`
						Pick    int `json:"pick"`
						Round   int `json:"round"`
						Team    struct {
							ID          interface{} `json:"id"`
							Abbr        string      `json:"abbreviation"`
							DisplayName string      `json:"displayName"`
						} `json:"team"`
						OriginalTeam struct {
							ID          interface{} `json:"id"`
							Abbr        string      `json:"abbreviation"`
							DisplayName string      `json:"displayName"`
						} `json:"originalTeam"`
						Selection struct {
							Athlete struct {
								DisplayName string `json:"displayName"`
								Position    struct {
									Abbr string `json:"abbreviation"`
								} `json:"position"`
								College struct {
									Name string `json:"name"`
								} `json:"college"`
							} `json:"athlete"`
						} `json:"selection"`
					} `json:"picks"`
				} `json:"rounds"`
			} `json:"draft"`
		} `json:"leagues"`
	} `json:"sports"`
}

func parseHeaderPayload(raw []byte) (domain.DraftSnapshot, error) {
	var dto headerDTO
	if err := json.Unmarshal(raw, &dto); err != nil {
		return domain.DraftSnapshot{}, err
	}
	if len(dto.Sports) == 0 || len(dto.Sports[0].Leagues) == 0 {
		return domain.DraftSnapshot{}, StructuredParseError{Kind: "unexpected_shape", Message: "header payload missing sports/leagues"}
	}
	draft := dto.Sports[0].Leagues[0].Draft
	if len(draft.Rounds) == 0 {
		return domain.DraftSnapshot{}, StructuredParseError{Kind: "unexpected_shape", Message: "header payload missing rounds"}
	}
	status := domain.DraftStatus{CurrentRound: draft.Status.CurrentRound, DraftActive: draft.Status.IsActive, State: "active"}
	if status.CurrentRound == 0 {
		status.CurrentRound = draft.Rounds[0].Round
	}
	picks := make([]domain.DraftPick, 0)
	for _, r := range draft.Rounds {
		for _, p := range r.Picks {
			overall := p.Overall
			if overall == 0 {
				overall = p.Pick
			}
			team := domain.Team{ID: toString(p.Team.ID), Abbr: p.Team.Abbr, DisplayName: p.Team.DisplayName}
			var original *domain.Team
			if p.OriginalTeam.DisplayName != "" || p.OriginalTeam.Abbr != "" || p.OriginalTeam.ID != nil {
				original = &domain.Team{ID: toString(p.OriginalTeam.ID), Abbr: p.OriginalTeam.Abbr, DisplayName: p.OriginalTeam.DisplayName}
			}
			player := p.Selection.Athlete.DisplayName
			pos := p.Selection.Athlete.Position.Abbr
			college := p.Selection.Athlete.College.Name
			picks = append(picks, domain.DraftPick{Overall: overall, Round: firstInt(p.Round, r.Round), CurrentTeam: team, OriginalTeam: original, PlayerName: player, Position: pos, College: college, SelectionFilled: player != ""})
		}
	}
	if len(picks) == 0 {
		return domain.DraftSnapshot{}, StructuredParseError{Kind: "unexpected_shape", Message: "header payload contains no picks"}
	}
	return domain.DraftSnapshot{Status: status, Picks: picks}, nil
}

func toString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.Itoa(int(x))
	case int:
		return strconv.Itoa(x)
	default:
		return ""
	}
}

// pickPlayerName returns the player name when "pick" is a JSON string and ""
// when it is a number, null or missing.
func pickPlayerName(raw json.RawMessage) string {
	var name string
	if err := json.Unmarshal(raw, &name); err != nil {
		return ""
	}
	return name
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func firstInt(a, b int) int {
	if a != 0 {
		return a
	}
	return b
}
