package asn

import (
	"context"
	"errors"
	"net/netip"
	"time"
)

var (
	ErrBlocked       = errors.New("asn blocked")
	ErrInvalidASNBan = errors.New("invalid asnban")
)

// Block represents a autonomous systems number based network block.
type Block struct {
	ASNum int `json:"as_num"`
	// Reason is the one liner reason shown to banned users upon connect.
	Reason string `json:"reason"`
	// Notes is the hidden moderator/admin notes for the ban.
	Notes     string    `json:"notes"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}

func NewBlock(asNum int, reason string) Block {
	return Block{ASNum: asNum, Reason: reason, CreatedOn: time.Now(), UpdatedOn: time.Now()}
}

func NewBlocker(repo Repository) Blocker {
	return Blocker{repo: repo}
}

type Blocker struct {
	repo Repository
}

func (a Blocker) Check(ctx context.Context, addr netip.Addr) error {
	if a.repo.IsBlocked(ctx, addr) {
		return ErrBlocked
	}

	return nil
}

func (a Blocker) Save(ctx context.Context, asnBan Block) error {
	if asnBan.ASNum <= 0 {
		return ErrInvalidASNBan
	}

	return a.repo.Save(ctx, asnBan)
}

func (a Blocker) Delete(ctx context.Context, asnBan Block) error {
	if asnBan.ASNum <= 0 {
		return ErrInvalidASNBan
	}

	return a.repo.Delete(ctx, asnBan.ASNum)
}
