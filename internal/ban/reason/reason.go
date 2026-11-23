package reason

// Reason defined a set of predefined ban reasons.
type Reason int

const (
	Custom Reason = iota + 1
	External
	Cheating
	Bigotry
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
		Bigotry:          "Bigotry",
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
