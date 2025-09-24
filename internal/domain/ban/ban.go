package ban

import "errors"

var ErrDuration = errors.New("invalid ban duration")

type SourceTarget struct {
	SourcePersonaname string `json:"source_personaname"`
	SourceAvatarhash  string `json:"source_avatarhash"`
	TargetPersonaname string `json:"target_personaname"`
	TargetAvatarhash  string `json:"target_avatarhash"`
}

// Type defines the state of the ban for a user, 0 being no ban.
type Type int

const (
	// Unknown means the ban state could not be determined, failing-open to allowing players
	// to connect.
	Unknown Type = iota - 1
	// OK Ban state is clean.
	OK //nolint:varnamelen
	// NoComm means the player cannot communicate while playing voice + chat.
	NoComm
	// Banned means the player cannot join the server at all.
	Banned
	// Network is used when a client connected from a banned CIDR block.
	Network
)

func (bt Type) String() string {
	switch bt {
	case Network:
		return "network"
	case Unknown:
		return "unknown"
	case NoComm:
		return "mute/gag"
	case Banned:
		return "banned"
	case OK:
		fallthrough
	default:
		return ""
	}
}

// Reason defined a set of predefined ban reasons.
type Reason int

const (
	Custom Reason = iota + 1
	External
	Cheating
	Racism
	Harassment
	Exploiting
	WarningsExceeded
	Spam
	Language
	Profile
	ItemDescriptions
	BotHost
	Evading
	Username
)

func (r Reason) String() string {
	return map[Reason]string{
		Custom:           "Custom",
		External:         "3rd party",
		Cheating:         "Cheating",
		Racism:           "Racism",
		Harassment:       "Personal Harassment",
		Exploiting:       "Exploiting",
		WarningsExceeded: "Warnings Exceeded",
		Spam:             "Spam",
		Language:         "Language",
		Profile:          "Profile",
		ItemDescriptions: "Item Name or Descriptions",
		BotHost:          "BotHost",
		Evading:          "Evading",
		Username:         "Inappropriate Username",
	}[r]
}

var Reasons = []Reason{ //nolint:gochecknoglobals
	External, Cheating, Racism, Harassment, Exploiting,
	WarningsExceeded, Spam, Language, Profile, ItemDescriptions,
	BotHost, Evading, Username, Custom,
}

// Origin defines the origin of the ban or action.
type Origin int

const (
	// System is an automatic ban triggered by the service.
	System Origin = iota
	// Bot is a ban using the discord bot interface.
	Bot
	// Web is a ban using the web-ui.
	Web
	// InGame is a ban using the sourcemod plugin.
	InGame
)

func (s Origin) String() string {
	switch s {
	case System:
		return "System"
	case Bot:
		return "Bot"
	case Web:
		return "Web"
	case InGame:
		return "In-Game"
	default:
		return "Unknown"
	}
}
