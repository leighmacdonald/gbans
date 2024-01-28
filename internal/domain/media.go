package domain

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

var MediaSafeMimeTypesImages = []string{
	"image/gif",
	"image/jpeg",
	"image/png",
	"image/webp",
}

type MediaRepository interface {
	SaveMedia(ctx context.Context, media *Media) error
	GetMediaByAssetID(ctx context.Context, uuid uuid.UUID, media *Media) error
	GetMediaByName(ctx context.Context, name string, media *Media) error
	GetMediaByID(ctx context.Context, mediaID int, media *Media) error
}

type MediaUsecase interface {
	Create(ctx context.Context, steamId steamid.SID64, name string, mimeType string, content []byte, mimeTypesAllowed []string) (*Media, error)
	GetMediaByAssetID(ctx context.Context, uuid uuid.UUID, media *Media) error
	GetMediaByName(ctx context.Context, name string, media *Media) error
	GetMediaByID(ctx context.Context, mediaID int, media *Media) error
}
