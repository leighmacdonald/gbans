package domain

import (
	"time"

	"github.com/leighmacdonald/steamid/v4/steamid"
)

type RequestUnban struct {
	UnbanReasonText string `json:"unban_reason_text"`
}

type SourceTarget struct {
	SourcePersonaname string `json:"source_personaname"`
	SourceAvatarhash  string `json:"source_avatarhash"`
	TargetPersonaname string `json:"target_personaname"`
	TargetAvatarhash  string `json:"target_avatarhash"`
}

type BanAppealMessage struct {
	BanID        int64           `json:"ban_id"`
	BanMessageID int64           `json:"ban_message_id"`
	AuthorID     steamid.SteamID `json:"author_id"`
	MessageMD    string          `json:"message_md"`
	Deleted      bool            `json:"deleted"`
	CreatedOn    time.Time       `json:"created_on"`
	UpdatedOn    time.Time       `json:"updated_on"`
	SimplePerson
}

func NewBanAppealMessage(banID int64, authorID steamid.SteamID, message string) BanAppealMessage {
	return BanAppealMessage{
		BanID:     banID,
		AuthorID:  authorID,
		MessageMD: message,
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
	}
}
