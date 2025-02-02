package domain

import (
	"time"

	"github.com/gofrs/uuid/v5"
)

type Message struct {
	ID              uuid.UUID `json:"id"`
	SteamID         string    `json:"steam_id"`
	CreatedOn       time.Time `json:"created_on"`
	Personaname     string    `json:"personaname"`
	Avatarhash      string    `json:"avatarhash"`
	PermissionLevel Privilege `json:"permission_level"`
	BodyMD          string    `json:"body_md"`
}
