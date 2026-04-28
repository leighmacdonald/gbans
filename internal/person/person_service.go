package person

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	v1 "github.com/leighmacdonald/gbans/internal/person/v1"
	"github.com/leighmacdonald/gbans/internal/person/v1/personv1connect"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PersonService struct {
	personv1connect.UnimplementedPersonServiceHandler

	persons *Persons
}

func NewPersonService(persons *Persons, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := personv1connect.NewPersonServiceHandler(PersonService{persons: persons}, option...)

	authMiddleware.UserRoute(personv1connect.PersonServiceProfileProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(personv1connect.PersonServiceResolveSteamIDProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(personv1connect.PersonServiceCurrentProfileProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(personv1connect.PersonServiceProfileSettingsProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(personv1connect.PersonServiceEditProfileSettingsProcedure, rpc.WithMinPermissions(permission.User))
	authMiddleware.UserRoute(personv1connect.PersonServiceQueryProcedure, rpc.WithMinPermissions(permission.Moderator))
	authMiddleware.UserRoute(personv1connect.PersonServiceEditPermissionsProcedure, rpc.WithMinPermissions(permission.Admin))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s PersonService) CurrentProfile(ctx context.Context, _ *emptypb.Empty) (*v1.CurrentProfileResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	requestCtx, cancelRequest := context.WithTimeout(ctx, time.Second*15)
	defer cancelRequest()

	response, err := s.persons.QueryProfile(requestCtx, user.SteamID.String())
	if err != nil {
		if errors.Is(err, steamid.ErrInvalidSID) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.CurrentProfileResponse{Profile: toPersonCore(response.Player)}, nil
}

func (s PersonService) Profile(ctx context.Context, req *v1.ProfileRequest) (*v1.ProfileResponse, error) {
	requestCtx, cancelRequest := context.WithTimeout(ctx, time.Second*15)
	defer cancelRequest()

	response, err := s.persons.QueryProfile(requestCtx, req.GetSteamId())
	if err != nil {
		if errors.Is(err, steamid.ErrInvalidSID) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.ProfileResponse{Profile: &v1.Profile{
		Player:   toPersonCore(response.Player),
		Friends:  toFriends(response.Friends),
		Settings: toSettings(response.Settings),
	}}, nil
}

func (s PersonService) ResolveSteamID(ctx context.Context, req *v1.ResolveSteamIDRequest) (*v1.ResolveSteamIDResponse, error) {
	requestCtx, cancelRequest := context.WithTimeout(ctx, time.Second*15)
	defer cancelRequest()

	response, err := s.persons.QueryProfile(requestCtx, req.GetSteamId())
	if err != nil {
		if errors.Is(err, steamid.ErrInvalidSID) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.ResolveSteamIDResponse{
		SteamId:     ptr.To(response.Player.SteamID.Int64()),
		AvatarHash:  ptr.To(string(response.Player.GetAvatar())),
		PersonaName: ptr.To(response.Player.GetName()),
	}, nil
}

func (s PersonService) ProfileSettings(ctx context.Context, _ *emptypb.Empty) (*v1.ProfileSettingsResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)

	settings, err := s.persons.GetPersonSettings(ctx, user.GetSteamID())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.ProfileSettingsResponse{Settings: toUserSettings(settings)}, nil
}

func (s PersonService) EditProfileSettings(ctx context.Context, req *v1.EditProfileSettingsRequest) (*v1.EditProfileSettingsResponse, error) {
	user, _ := rpc.UserInfoFromCtx(ctx)
	settings, err := s.persons.SavePersonSettings(ctx, user, SettingsUpdate{
		ForumSignature:       req.GetForumSignature(),
		ForumProfileMessages: req.GetForumProfileMessages(),
		StatsHidden:          req.GetStatsHidden(),
		CenterProjectiles:    req.CenterProjectiles,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.EditProfileSettingsResponse{Settings: toUserSettings(settings)}, nil
}

func (s PersonService) Query(ctx context.Context, req *v1.QueryRequest) (*v1.QueryResponse, error) {
	var perms []permission.Privilege
	for _, perm := range req.GetWithPermissions() {
		perms = append(perms, permission.Privilege(perm))
	}
	query := Query{
		Filter:            rpc.FromRPC(req.GetFilter()),
		PersonaName:       req.GetPersonaName(),
		WithPermissions:   perms,
		DiscordID:         req.GetDiscordId(),
		SteamIDs:          req.SteamIds,
		VacBans:           req.GetVacBans(),
		GameBans:          req.GetGameBans(),
		AvatarHash:        req.GetAvatarHash(),
		CommunityBanned:   ptr.To(req.GetCommunityBanned()),
		TimeCreatedAfter:  ptr.To(req.TimeCreatedAfter.AsTime()),
		TimeCreatedBefore: ptr.To(req.TimeCreatedBefore.AsTime()),
	}
	people, count, errGetPeople := s.persons.GetPeople(ctx, query)
	if errGetPeople != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	resp := v1.QueryResponse{Count: &count, People: make([]*v1.Person, len(people))}
	for idx, person := range people {
		resp.People[idx] = toPerson(&person)
	}

	return &resp, nil
}

func (s PersonService) EditPermissions(ctx context.Context, req *v1.EditPermissionsRequest) (*v1.EditPermissionsResponse, error) {
	player, errPerson := s.persons.BySteamID(ctx, steamid.New(req.GetSteamId()))
	if errPerson != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	player.PermissionLevel = permission.Privilege(req.GetPermissionLevel())

	if err := s.persons.Save(ctx, &player); err != nil {
		if errors.Is(err, permission.ErrDenied) {
			return nil, connect.NewError(connect.CodePermissionDenied, rpc.ErrPermission)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	slog.Info("Player permission updated",
		slog.Int64("steam_id", player.SteamID.Int64()),
		slog.String("permissions", req.PermissionLevel.String()))

	return &v1.EditPermissionsResponse{Person: toPersonCore(&player)}, nil
}

func toUserSettings(settings Settings) *v1.UserSettings {
	return &v1.UserSettings{
		PersonSettingsId:     &settings.PersonSettingsID,
		SteamId:              ptr.To(settings.SteamID.Int64()),
		ForumSignature:       &settings.ForumSignature,
		ForumProfileMessages: &settings.ForumProfileMessages,
		StatsHidden:          &settings.StatsHidden,
		CreatedOn:            timestamppb.New(settings.CreatedOn),
		UpdatedOn:            timestamppb.New(settings.UpdatedOn),
	}
}

func toPersonCore(core *Person) *v1.PersonCore {
	return &v1.PersonCore{
		SteamId:         ptr.To(core.SteamID.Int64()),
		PermissionLevel: ptr.To(v1.Privilege(core.PermissionLevel)),
		Name:            ptr.To(core.GetName()),
		AvatarHash:      ptr.To(string(core.GetAvatar())),
		DiscordId:       ptr.To(core.GetDiscordID()),
		VacBans:         ptr.To(core.GetVACBans()),
		GameBans:        ptr.To(core.GetGameBans()),
		TimeCreated:     timestamppb.New(core.GetTimeCreated()),
	}
}

func toPerson(core *Person) *v1.Person {
	var tsBB *timestamppb.Timestamp
	if core.LastLogoff != nil {
		tsBB = timestamppb.New(*core.LastLogoff)
	}

	return &v1.Person{
		SteamId:               ptr.To(core.SteamID.Int64()),
		CreatedOn:             timestamppb.New(core.CreatedOn),
		UpdatedOn:             timestamppb.New(core.UpdatedOn),
		PermissionLevel:       ptr.To(v1.Privilege(core.PermissionLevel)),
		Muted:                 &core.Muted,
		DiscordId:             ptr.To(core.GetDiscordID()),
		PatreonId:             &core.PatreonID,
		IpAddr:                ptr.To(core.IPAddr.String()),
		CommunityBanned:       &core.CommunityBanned,
		VacBans:               ptr.To(core.GetVACBans()),
		GameBans:              ptr.To(core.GetGameBans()),
		EconomyBan:            ptr.To(string(core.EconomyBan)),
		DaysSinceLastBan:      &core.DaysSinceLastBan,
		UpdatedOnSteam:        timestamppb.New(core.UpdatedOnSteam),
		PlayerqueueChatStatus: nil,
		PlayerqueueChatReason: nil,
		AvatarHash:            ptr.To(string(core.GetAvatar())),
		CommentPermission:     &core.CommentPermission,
		LastLogoff:            tsBB,
		LocCityId:             &core.LocCityID,
		LocCountryCode:        &core.LocCountryCode,
		LocStateCode:          &core.LocStateCode,
		PersonaName:           ptr.To(core.GetName()),
		PersonaState:          &core.PersonaState,
		PersonaStateFlags:     &core.PersonaStateFlags,
		PrimaryClanId:         &core.PrimaryClanID,
		ProfileState:          &core.ProfileState,
		ProfileUrl:            &core.ProfileURL,
		RealName:              &core.RealName,
		TimeCreated:           timestamppb.New(core.GetTimeCreated()),
		VisibilityState:       ptr.To(v1.VisibilityState(core.VisibilityState)),
		BanId:                 nil,
	}
}

func toFriends(friendSet []thirdparty.SteamFriend) []*v1.SteamFriend {
	friends := make([]*v1.SteamFriend, len(friendSet))
	for idx, friend := range friendSet {
		sid := steamid.New(friend.SteamId)
		friends[idx] = &v1.SteamFriend{
			FriendSince:  timestamppb.New(friend.FriendSince),
			Relationship: &friend.Relationship,
			RemovedOn:    timestamppb.New(friend.RemovedOn),
			SteamId:      ptr.To(sid.Int64()),
		}
	}

	return friends
}

func toSettings(settings Settings) *v1.Settings {
	return &v1.Settings{
		PersonSettingsId:     &settings.PersonSettingsID,
		SteamId:              ptr.To(settings.SteamID.Int64()),
		ForumSignature:       &settings.ForumSignature,
		ForumProfileMessages: &settings.ForumProfileMessages,
		StatsHidden:          &settings.StatsHidden,
		CenterProjectiles:    settings.CenterProjectiles,
		CreatedOn:            timestamppb.New(settings.CreatedOn),
		UpdatedOn:            timestamppb.New(settings.UpdatedOn),
	}
}
