package appeal

import "github.com/leighmacdonald/gbans/internal/ban"

type AppealOverview struct {
	ban.Ban

	SourcePersonaname string `json:"source_personaname"`
	SourceAvatarhash  string `json:"source_avatarhash"`
	TargetPersonaname string `json:"target_personaname"`
	TargetAvatarhash  string `json:"target_avatarhash"`
}
