package model

type Privilege uint8

const (
	PBanned    Privilege = 0 // Logged in, but is banned
	PUser      Privilege = 1
	PModerator Privilege = 50
	PAdmin     Privilege = 100
)
