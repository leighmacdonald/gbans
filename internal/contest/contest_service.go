package contest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/asset"
	assetv1 "github.com/leighmacdonald/gbans/internal/asset/v1"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	v1 "github.com/leighmacdonald/gbans/internal/contest/v1"
	"github.com/leighmacdonald/gbans/internal/contest/v1/contestv1connect"
	"github.com/leighmacdonald/gbans/internal/database"
	personv1 "github.com/leighmacdonald/gbans/internal/person/v1"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	contestv1connect.UnimplementedServiceHandler

	contests Contests
	assets   asset.Assets
}

func NewService(contests Contests, assets asset.Assets) Service {
	return Service{contests: contests, assets: assets}
}

func (s Service) Contests(ctx context.Context, _ *emptypb.Empty) (*v1.ContestsResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	contests, errContests := s.contests.Contests(ctx, user)
	if errContests != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.ContestsResponse{Contests: make([]*v1.Contest, len(contests))}
	for idx, contest := range contests {
		resp.Contests[idx] = toContest(contest)
	}

	return &resp, nil
}

func (s Service) Contest(ctx context.Context, req *v1.ContestRequest) (*v1.ContestResponse, error) {
	contestID, _ := uuid.FromString(req.GetContestId())
	var contest Contest
	if err := s.contests.ByID(ctx, contestID, &contest); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
	}

	return &v1.ContestResponse{Contest: toContest(contest)}, nil
}

func (s Service) Entries(ctx context.Context, req *v1.EntriesRequest) (*v1.EntriesResponse, error) {
	contestID, _ := uuid.FromString(req.GetContestId())
	entries, errEntries := s.contests.Entries(ctx, contestID)
	if errEntries != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.EntriesResponse{Entries: make([]*v1.Entry, len(entries))}
	for idx, entry := range entries {
		resp.Entries[idx] = toEntry(entry)
	}

	return &resp, nil
}

func (s Service) Upload(ctx context.Context, req *v1.UploadRequest) (*v1.UploadResponse, error) {
	contestId, _ := uuid.FromString(req.GetContestId())
	var contest Contest
	if errContest := s.contests.ByID(ctx, contestId, &contest); errContest != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	body := req.GetContent()
	if contest.MediaTypes != "" {
		mimeType, errMimeType := mimetype.DetectReader(bytes.NewReader(body))
		if errMimeType != nil {
			return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
		}

		if !slices.Contains(strings.Split(strings.ToLower(contest.MediaTypes), ","), strings.ToLower(mimeType.String())) {
			return nil, connect.NewError(connect.CodeInvalidArgument, rpc.ErrBadRequest)
		}
	}

	user, _ := rpc.UserInfoFromCtx(ctx)
	mediaAsset, errCreate := s.assets.Create(ctx, user.GetSteamID(), "media", req.GetName(), bytes.NewReader(body), false)
	if errCreate != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.UploadResponse{Asset: toAsset(mediaAsset)}, nil
}

func toAsset(asset asset.Asset) *assetv1.Asset {
	return &assetv1.Asset{
		AssetId:   ptr.To(asset.AssetID.String()),
		Bucket:    ptr.To(string(asset.Bucket)),
		AuthorId:  ptr.To(asset.AuthorID.Int64()),
		Hash:      ptr.To(fmt.Sprintf("%x", asset.Hash)),
		IsPrivate: &asset.IsPrivate,
		MimeType:  &asset.MimeType,
		Name:      &asset.Name,
		Size:      &asset.Size,
		CreatedOn: timestamppb.New(asset.CreatedOn),
		UpdatedOn: timestamppb.New(asset.UpdatedOn),
	}
}

func (s Service) Vote(ctx context.Context, req *v1.VoteRequest) (*v1.VoteResponse, error) {
	contestID, _ := uuid.FromString(req.GetContestId())
	contestEntryID, _ := uuid.FromString(req.GetContestEntryId())
	direction := req.GetDirection()

	user, _ := rpc.UserInfoFromCtx(ctx)
	if errVote := s.contests.EntryVote(ctx, contestID, contestEntryID, user, direction == v1.Direction_DIRECTION_UP_UNSPECIFIED); errVote != nil {
		if !errors.Is(errVote, ErrVoteDeleted) {
			return &v1.VoteResponse{}, nil
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.VoteResponse{CurrentDirection: &direction}, nil
}

func (s Service) EntryCreate(ctx context.Context, req *v1.EntryCreateRequest) (*v1.EntryCreateResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	contestID, _ := uuid.FromString(req.GetContestId())
	assetID, _ := uuid.FromString(req.GetAssetId())
	var contest Contest
	if errContest := s.contests.ByID(ctx, contestID, &contest); errContest != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	existingEntries, errEntries := s.contests.Entries(ctx, contest.ContestID)
	if errEntries != nil && !errors.Is(errEntries, database.ErrNoResult) {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	own := int32(0)
	for _, entry := range existingEntries {
		if entry.SteamID == user.GetSteamID() {
			own++
		}

		if own >= contest.MaxSubmissions {
			return nil, connect.NewError(connect.CodeAlreadyExists, rpc.ErrPermission)
			// httphelper.SetError(ctx, httphelper.NewAPIErrorf(http.StatusForbidden, ErrContestMaxEntries,
			//	"You have already submitted the max (%d) allowable items.", contest.MaxSubmissions))
		}
	}

	existingAsset, errAsset := s.assets.Get(ctx, assetID)
	if errAsset != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	if existingAsset.AuthorID != user.GetSteamID() {
		return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
	}

	entry, errEntry := contest.NewEntry(user.GetSteamID(), assetID, req.GetDescription())
	if errEntry != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	if errSave := s.contests.EntrySave(ctx, entry); errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	slog.Info("New contest entry submitted", slog.String("contest_id", contest.ContestID.String()))

	return &v1.EntryCreateResponse{Entry: toEntry(&entry)}, nil
}

func (s Service) EntryDelete(ctx context.Context, req *v1.EntryDeleteRequest) (*emptypb.Empty, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	contestEntryID, _ := uuid.FromString(req.GetContestEntryId())
	var entry Entry
	if errContest := s.contests.Entry(ctx, contestEntryID, &entry); errContest != nil {
		if errors.Is(errContest, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	// Only >=moderators or the entry author are allowed to delete entries.
	if !user.HasPermission(permission.Moderator) || user.GetSteamID() != entry.SteamID {
		return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
	}

	var contest Contest
	if errContest := s.contests.ByID(ctx, entry.ContestID, &contest); errContest != nil {
		if errors.Is(errContest, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	// Only allow mods to delete entries from expired contests.
	if user.GetSteamID() == entry.SteamID && time.Since(contest.DateEnd) > 0 {
		return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
	}

	if errDelete := s.contests.EntryDelete(ctx, entry.ContestEntryID); errDelete != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}
	slog.Info("Contest deleted",
		slog.String("contest_id", entry.ContestID.String()),
		slog.String("contest_entry_id", entry.ContestEntryID.String()),
		slog.String("title", contest.Title))

	return &emptypb.Empty{}, nil
}

func (s Service) ContestCreate(ctx context.Context, req *v1.ContestCreateRequest) (*v1.ContestCreateResponse, error) {
	contest, errSave := s.contests.Save(ctx, fromContest(req.Contest))
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.ContestCreateResponse{Contest: toContest(contest)}, nil
}

func (s Service) ContestDelete(ctx context.Context, req *v1.ContestDeleteRequest) (*emptypb.Empty, error) {
	contestID, _ := uuid.FromString(req.GetContestId())
	var contest Contest

	if errContest := s.contests.ByID(ctx, contestID, &contest); errContest != nil {
		if errors.Is(errContest, database.ErrNoResult) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	if errDelete := s.contests.ContestDelete(ctx, contest.ContestID); errDelete != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &emptypb.Empty{}, nil
}

func (s Service) ContestEdit(ctx context.Context, req *v1.ContestEditRequest) (*v1.ContestEditResponse, error) {
	// FIXME check for contest.
	contest, errSave := s.contests.Save(ctx, fromContest(req.GetContest()))
	if errSave != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.ContestEditResponse{Contest: toContest(contest)}, nil
}

func fromContest(contest *v1.Contest) Contest {
	contestID, _ := uuid.FromString(contest.GetContestId())

	return Contest{
		CreatedOn:          contest.GetCreatedOn().AsTime(),
		UpdatedOn:          contest.GetUpdatedOn().AsTime(),
		ContestID:          contestID,
		Title:              contest.GetTitle(),
		Description:        contest.GetDescription(),
		Public:             contest.GetPublic(),
		HideSubmissions:    contest.GetHideSubmissions(),
		DateStart:          contest.DateStart.AsTime(),
		DateEnd:            contest.DateEnd.AsTime(),
		MaxSubmissions:     contest.GetMaxSubmissions(),
		OwnSubmissions:     contest.GetOwnSubmissions(),
		MediaTypes:         contest.GetMediaTypes(),
		NumEntries:         contest.GetNumEntries(),
		Deleted:            false,
		Voting:             contest.GetVoting(),
		MinPermissionLevel: permission.Privilege(contest.GetMinPermissionLevel()),
		DownVotes:          contest.GetDownVotes(),
		IsNew:              false,
	}
}

func toContest(contest Contest) *v1.Contest {
	return &v1.Contest{
		Title:              &contest.Title,
		Description:        &contest.Description,
		Public:             &contest.Public,
		HideSubmissions:    &contest.HideSubmissions,
		DateStart:          timestamppb.New(contest.DateStart),
		DateEnd:            timestamppb.New(contest.DateEnd),
		MaxSubmissions:     &contest.MaxSubmissions,
		OwnSubmissions:     &contest.OwnSubmissions,
		MediaTypes:         &contest.MediaTypes,
		NumEntries:         &contest.NumEntries,
		Voting:             &contest.Voting,
		MinPermissionLevel: ptr.To(personv1.Privilege(contest.MinPermissionLevel)),
		DownVotes:          &contest.DownVotes,
		CreatedOn:          timestamppb.New(contest.CreatedOn),
		UpdatedOn:          timestamppb.New(contest.UpdatedOn),
		ContestId:          nil,
	}
}

func toEntry(entry *Entry) *v1.Entry {
	return &v1.Entry{
		ContestId:      ptr.To(entry.ContestID.String()),
		ContestEntryId: ptr.To(entry.ContestEntryID.String()),
		SteamId:        ptr.To(entry.SteamID.Int64()),
		PersonaName:    &entry.Personaname,
		AvatarHash:     &entry.AvatarHash,
		AssetId:        ptr.To(entry.AssetID.String()),
		Description:    &entry.Description,
		Placement:      &entry.Placement,
		Deleted:        &entry.Deleted,
		VotesUp:        &entry.VotesUp,
		VotesDown:      &entry.VotesDown,
		Asset:          toAsset(entry.Asset),
		CreatedOn:      timestamppb.New(entry.CreatedOn),
		UpdatedOn:      timestamppb.New(entry.UpdatedOn),
	}
}
