package discord

type Config struct {
	Enabled                 bool   `json:"enabled"`
	BotEnabled              bool   `json:"bot_enabled"`
	IntegrationsEnabled     bool   `json:"integrations_enabled"`
	AppID                   string `json:"app_id"`
	AppSecret               string `json:"app_secret"`
	LinkID                  string `json:"link_id"`
	Token                   string `json:"token"`
	GuildID                 string `json:"guild_id"`
	PublicLogChannelEnable  bool   `json:"public_log_channel_enable"`
	LogChannelID            string `json:"log_channel_id"`
	PublicLogChannelID      string `json:"public_log_channel_id"`
	PublicMatchLogChannelID string `json:"public_match_log_channel_id"`
	VoteLogChannelID        string `json:"vote_log_channel_id"`
	AppealLogChannelID      string `json:"appeal_log_channel_id"`
	BanLogChannelID         string `json:"ban_log_channel_id"`
	ForumLogChannelID       string `json:"forum_log_channel_id"`
	KickLogChannelID        string `json:"kick_log_channel_id"`
	PlayerqueueChannelID    string `json:"playerqueue_channel_id"`
	ModPingRoleID           string `json:"mod_ping_role_id"`
	AnticheatChannelID      string `json:"anticheat_channel_id"`
	SeedChannelID           string `json:"seed_channel_id"`
}

func (c Config) SafePublicLogChannelID() string {
	if c.PublicLogChannelID != "" {
		return c.PublicLogChannelID
	}

	return c.LogChannelID
}

func (c Config) SafeSeedChannelID() string {
	if c.SeedChannelID != "" {
		return c.SeedChannelID
	}

	return c.LogChannelID
}

func (c Config) SafePublicMatchLogChannelID() string {
	if c.PublicMatchLogChannelID != "" {
		return c.PublicMatchLogChannelID
	}

	return c.LogChannelID
}

func (c Config) SafeVoteLogChannelID() string {
	if c.VoteLogChannelID != "" {
		return c.VoteLogChannelID
	}

	return c.LogChannelID
}

func (c Config) SafeAppealLogChannelID() string {
	if c.AppealLogChannelID != "" {
		return c.AppealLogChannelID
	}

	return c.LogChannelID
}

func (c Config) SafeBanLogChannelID() string {
	if c.BanLogChannelID != "" {
		return c.BanLogChannelID
	}

	return c.LogChannelID
}

func (c Config) SafeForumLogChannelID() string {
	if c.ForumLogChannelID != "" {
		return c.ForumLogChannelID
	}

	return c.LogChannelID
}

func (c Config) SafeKickLogChannelID() string {
	if c.KickLogChannelID != "" {
		return c.KickLogChannelID
	}

	return c.LogChannelID
}

func (c Config) SafePlayerqueueChannelID() string {
	if c.PlayerqueueChannelID != "" {
		return c.PlayerqueueChannelID
	}

	return c.LogChannelID
}

func (c Config) SafeModPingRoleID() string {
	if c.ModPingRoleID != "" {
		return c.ModPingRoleID
	}

	return c.LogChannelID
}

func (c Config) SafeAnticheatChannelID() string {
	if c.AnticheatChannelID != "" {
		return c.AnticheatChannelID
	}

	return c.LogChannelID
}
