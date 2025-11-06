package thirdparty

type FileInfo struct {
	Authors     []string `json:"authors"`
	Description string   `json:"description"`
	Title       string   `json:"title"`
	UpdateURL   string   `json:"update_url"`
}

type LastSeen struct {
	PlayerName string `json:"player_name,omitempty"`
	Time       int    `json:"time,omitempty"`
}

type Players struct {
	Attributes []string `json:"attributes"`
	LastSeen   LastSeen `json:"last_seen"`
	Steamid    any      `json:"steamid"`
	Proof      []string `json:"proof,omitempty"`
}

type TF2BDSchema struct {
	Schema   string    `json:"$schema"` //nolint:tagliatelle
	FileInfo FileInfo  `json:"file_info"`
	Players  []Players `json:"players"`
}
