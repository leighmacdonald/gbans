package person

import "fmt"

const (
	avatarURLSmallFormat  = "https://avatars.akamai.steamstatic.com/%s.jpg"
	avatarURLMediumFormat = "https://avatars.akamai.steamstatic.com/%s_medium.jpg"
	avatarURLFullFormat   = "https://avatars.akamai.steamstatic.com/%s_full.jpg"
)

func NewAvatarLinks(hash string) AvatarLinks {
	return AvatarLinks{hash: hash}
}

type AvatarLinks struct {
	hash string
}

func (h AvatarLinks) Full() string {
	return fmt.Sprintf(avatarURLFullFormat, h.hash)
}

func (h AvatarLinks) Medium() string {
	return fmt.Sprintf(avatarURLMediumFormat, h.hash)
}

func (h AvatarLinks) Small() string {
	return fmt.Sprintf(avatarURLSmallFormat, h.hash)
}
