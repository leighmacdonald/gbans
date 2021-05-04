package model

type Privilege uint8

const (
	PGuest         Privilege = 1
	PBanned        Privilege = 2 // Logged in, but is banned
	PAuthenticated Privilege = 10
	PModerator     Privilege = 50
	PAdmin         Privilege = 100
)
