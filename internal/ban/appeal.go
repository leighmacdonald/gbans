package ban

type NewBanMessage struct {
	Message string `json:"message"`
}

type AppealQueryFilter struct {
	Deleted bool `json:"deleted"`
}
