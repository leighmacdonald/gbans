package permission

import "errors"

var ErrDenied = errors.New("permission denied")

type Privilege uint8

const (
	Banned    Privilege = 0   // Logged in, but is banned
	Guest     Privilege = 1   // Non logged in user
	User      Privilege = 10  // Normal logged-in user
	Reserved  Privilege = 15  // Normal logged-in user with reserved slot
	Editor    Privilege = 25  // Edit Access to site / resources (not really used yet)
	Moderator Privilege = 50  // Access detailed player into & ban permissions.
	Admin     Privilege = 100 // Unrestricted admin
)

func (p Privilege) String() string {
	switch p {
	case Banned:
		return "banned"
	case Guest:
		return "guest"
	case User:
		return "user"
	case Reserved:
		return "reserved"
	case Editor:
		return "editor"
	case Moderator:
		return "moderator"
	case Admin:
		return "admin"
	default:
		return "unknown"
	}
}
