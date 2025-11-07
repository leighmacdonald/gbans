package bantype

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
