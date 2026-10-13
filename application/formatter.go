package application

import (
	"fmt"

	"github.com/Mundo-Dolphins/dolphins-draft-bot/domain"
)

const defaultTeamName = "Miami Dolphins"

// MessageFormatter renders the Telegram messages. TeamName defaults to
// "Miami Dolphins" when empty.
type MessageFormatter struct {
	TeamName string
}

func (f MessageFormatter) team() string {
	if f.TeamName == "" {
		return defaultTeamName
	}
	return f.TeamName
}

func (f MessageFormatter) Selection(p domain.DraftPick) string {
	return fmt.Sprintf("The pick is in, %s han seleccionado con el pick %d a %s, %s, %s", f.team(), p.Overall, p.PlayerName, p.Position, p.College)
}

func (f MessageFormatter) TradeOut(overall int, team string) string {
	return fmt.Sprintf("Trade alert: el pick %d de %s ha sido traspasado a %s", overall, f.team(), team)
}

func (f MessageFormatter) TradeIn(overall int, source string) string {
	if source == "" {
		return fmt.Sprintf("Trade alert: %s ha adquirido el pick %d", f.team(), overall)
	}
	return fmt.Sprintf("Trade alert: %s ha adquirido el pick %d, originalmente de %s", f.team(), overall, source)
}
