package discord

import "sync"

type Config struct {
	sync.RWMutex

	Enabled                 bool
	BotEnabled              bool
	IntegrationsEnabled     bool
	AppID                   string
	AppSecret               string
	LinkID                  string
	Token                   string
	GuildID                 string
	PublicLogChannelEnable  bool
	LogChannelID            string
	PublicLogChannelID      string
	PublicMatchLogChannelID string
	VoteLogChannelID        string
	AppealLogChannelID      string
	BanLogChannelID         string
	ForumLogChannelID       string
	KickLogChannelID        string
	ModPingRoleID           string
	AnticheatChannelID      string
	SeedChannelID           string
	WordFilterLogChannelID  string
	ChatLogChannelID        string

	// Unregister UnregisterOnClose when true, will unregister all the previously registered commands on shutdown.
	UnregisterOnClose bool
	// UserAgent defines a optional custom user agent.
	UserAgent string
}

func (c *Config) SafePublicLogChannelID() string {
	if c.PublicLogChannelID != "" {
		return c.PublicLogChannelID
	}

	return c.LogChannelID
}

func (c *Config) SafeSeedChannelID() string {
	if c.SeedChannelID != "" {
		return c.SeedChannelID
	}

	return c.LogChannelID
}

func (c *Config) SafeVoteLogChannelID() string {
	if c.VoteLogChannelID != "" {
		return c.VoteLogChannelID
	}

	return c.LogChannelID
}

func (c *Config) SafeAppealLogChannelID() string {
	if c.AppealLogChannelID != "" {
		return c.AppealLogChannelID
	}

	return c.LogChannelID
}

func (c *Config) SafeBanLogChannelID() string {
	if c.BanLogChannelID != "" {
		return c.BanLogChannelID
	}

	return c.LogChannelID
}

func (c *Config) SafeForumLogChannelID() string {
	if c.ForumLogChannelID != "" {
		return c.ForumLogChannelID
	}

	return c.LogChannelID
}

func (c *Config) SafeKickLogChannelID() string {
	if c.KickLogChannelID != "" {
		return c.KickLogChannelID
	}

	return c.LogChannelID
}

func (c *Config) SafeModPingRoleID() string {
	if c.ModPingRoleID != "" {
		return c.ModPingRoleID
	}

	return c.LogChannelID
}

func (c *Config) SafeAnticheatChannelID() string {
	if c.AnticheatChannelID != "" {
		return c.AnticheatChannelID
	}

	return c.LogChannelID
}

func (c *Config) SafeWordFilterLogChannelID() string {
	if c.WordFilterLogChannelID != "" {
		return c.WordFilterLogChannelID
	}

	return c.LogChannelID
}

func (c *Config) SafeChatLogChannelID() string {
	if c.ChatLogChannelID != "" {
		return c.ChatLogChannelID
	}

	return c.LogChannelID
}
