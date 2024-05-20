package domain

type NotificationSeverity int

const (
	SeverityInfo NotificationSeverity = iota
	SeverityWarn
	SeverityError
)

type Privilege uint8

const (
	PBanned    Privilege = 0   // Logged in, but is banned
	PGuest     Privilege = 1   // Non logged in user
	PUser      Privilege = 10  // Normal logged-in user
	PReserved  Privilege = 15  // Normal logged-in user with reserved slot
	PEditor    Privilege = 25  // Edit Access to site / resources
	PModerator Privilege = 50  // Access detailed player into & ban permissions.
	PAdmin     Privilege = 100 // Unrestricted admin
)

func (p Privilege) String() string {
	switch p {
	case PBanned:
		return "banned"
	case PGuest:
		return "guest"
	case PUser:
		return "user"
	case PReserved:
		return "reserved"
	case PEditor:
		return "editor"
	case PModerator:
		return "moderator"
	case PAdmin:
		return "admin"
	default:
		return "unknown"
	}
}
