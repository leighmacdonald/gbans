package person

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/leighmacdonald/gbans/internal/person/v1"
	"github.com/leighmacdonald/gbans/internal/person/v1/personv1connect"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/gbans/internal/thirdparty"
	"github.com/leighmacdonald/steamid/v4/steamid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	// personv1connect.UnimplementedPersonServiceHandler

	persons  *Persons
	roleAuth *rpc.RoleAuth
}

func NewPersonService(persons *Persons, roleAuth *rpc.RoleAuth, authMiddleware *rpc.Middleware, option ...connect.HandlerOption) rpc.Service {
	pattern, handler := personv1connect.NewPersonServiceHandler(Service{persons: persons, roleAuth: roleAuth}, option...)

	authMiddleware.UserRoute(personv1connect.PersonServiceProfileProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_PERSON_READ))
	authMiddleware.UserRoute(personv1connect.PersonServiceResolveSteamIDProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_STEAMID_RESOLVE))
	authMiddleware.AuthedRoute(personv1connect.PersonServiceCurrentProfileProcedure)
	authMiddleware.UserRoute(personv1connect.PersonServiceProfileSettingsProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_CURRENT_SETTINGS))
	authMiddleware.UserRoute(personv1connect.PersonServiceEditProfileSettingsProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_CURRENT_SETTINGS))
	authMiddleware.UserRoute(personv1connect.PersonServiceQueryProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_PERSON_READ))

	return rpc.Service{Pattern: pattern, Handler: handler}
}

func (s Service) CurrentProfile(ctx context.Context, _ *emptypb.Empty) (*v1.CurrentProfileResponse, error) {
	user := rpc.UserInfoFromCtx(ctx)
	requestCtx, cancelRequest := context.WithTimeout(ctx, time.Second*15)
	defer cancelRequest()

	response, err := s.persons.QueryProfile(requestCtx, user.SteamID.String())
	if err != nil {
		if errors.Is(err, steamid.ErrInvalidSID) {
			return nil, connect.NewError(connect.CodeNotFound, rpc.ErrNotFound)
		}

		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	core := toPersonCore(response.Player)
	core.Permissions = s.roleAuth.PermissionsBySteamID(ctx, user.GetSteamID())

	return &v1.CurrentProfileResponse{Profile: core}, nil
}

func (s Service) Profile(ctx context.Context, req *v1.ProfileRequest) (*v1.ProfileResponse, error) {
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

func (s Service) ResolveSteamID(ctx context.Context, req *v1.ResolveSteamIDRequest) (*v1.ResolveSteamIDResponse, error) {
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
		SteamId:     new(response.Player.SteamID.Int64()),
		AvatarHash:  new(string(response.Player.GetAvatar())),
		PersonaName: new(response.Player.GetName()),
	}, nil
}

func (s Service) ProfileSettings(ctx context.Context, _ *emptypb.Empty) (*v1.ProfileSettingsResponse, error) {
	user := rpc.UserInfoFromCtx(ctx)

	settings, err := s.persons.GetPersonSettings(ctx, user.GetSteamID())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, rpc.ErrInternal)
	}

	return &v1.ProfileSettingsResponse{Settings: toUserSettings(settings)}, nil
}

func (s Service) EditProfileSettings(ctx context.Context, req *v1.EditProfileSettingsRequest) (*v1.EditProfileSettingsResponse, error) {
	user := rpc.UserInfoFromCtx(ctx)
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

func (s Service) Query(ctx context.Context, req *v1.QueryRequest) (*v1.QueryResponse, error) {
	query := Query{
		Filter:            rpc.FromRPC(req.GetFilter()),
		PersonaName:       req.GetPersonaName(),
		DiscordID:         req.GetDiscordId(),
		SteamIDs:          req.GetSteamIds(),
		VacBans:           req.GetVacBans(),
		GameBans:          req.GetGameBans(),
		AvatarHash:        req.GetAvatarHash(),
		CommunityBanned:   new(req.GetCommunityBanned()),
		TimeCreatedAfter:  new(req.GetTimeCreatedAfter().AsTime()),
		TimeCreatedBefore: new(req.GetTimeCreatedBefore().AsTime()),
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

func toUserSettings(settings Settings) *v1.Settings {
	return &v1.Settings{
		PersonSettingsId:     &settings.PersonSettingsID,
		SteamId:              new(settings.SteamID.Int64()),
		ForumSignature:       &settings.ForumSignature,
		ForumProfileMessages: &settings.ForumProfileMessages,
		StatsHidden:          &settings.StatsHidden,
		CreatedOn:            timestamppb.New(settings.CreatedOn),
		UpdatedOn:            timestamppb.New(settings.UpdatedOn),
	}
}

func toPersonCore(core *Person) *v1.PersonCore {
	return &v1.PersonCore{
		SteamId:     new(core.SteamID.Int64()),
		Permissions: core.Permissions,
		Name:        new(core.GetName()),
		AvatarHash:  new(string(core.GetAvatar())),
		DiscordId:   new(core.GetDiscordID()),
		VacBans:     new(core.GetVACBans()),
		GameBans:    new(core.GetGameBans()),
		TimeCreated: timestamppb.New(core.GetTimeCreated()),
		BanId:       &core.BanID,
		PatreonId:   &core.PatreonID,
	}
}

func stringsToPermissions(perms []string) []rolesv1.Permission {
	out := make([]rolesv1.Permission, 0, len(perms))

	for _, s := range perms {
		p, ok := rolesv1.Permission_value[s]
		if !ok || p == 0 {
			continue
		}

		out = append(out, rolesv1.Permission(p))
	}

	if out == nil {
		return []rolesv1.Permission{}
	}

	return out
}

func toPerson(core *Person) *v1.Person {
	var tsBB *timestamppb.Timestamp
	if core.LastLogoff != nil {
		tsBB = timestamppb.New(*core.LastLogoff)
	}

	return &v1.Person{
		SteamId:               new(core.SteamID.Int64()),
		CreatedOn:             timestamppb.New(core.CreatedOn),
		UpdatedOn:             timestamppb.New(core.UpdatedOn),
		Muted:                 &core.Muted,
		DiscordId:             new(core.GetDiscordID()),
		PatreonId:             &core.PatreonID,
		IpAddr:                new(core.IPAddr.String()),
		CommunityBanned:       &core.CommunityBanned,
		VacBans:               new(core.GetVACBans()),
		GameBans:              new(core.GetGameBans()),
		EconomyBan:            new(string(core.EconomyBan)),
		DaysSinceLastBan:      &core.DaysSinceLastBan,
		UpdatedOnSteam:        timestamppb.New(core.UpdatedOnSteam),
		PlayerqueueChatStatus: nil,
		PlayerqueueChatReason: nil,
		AvatarHash:            new(string(core.GetAvatar())),
		CommentPermission:     &core.CommentPermission,
		LastLogoff:            tsBB,
		LocCityId:             &core.LocCityID,
		LocCountryCode:        &core.LocCountryCode,
		LocStateCode:          &core.LocStateCode,
		PersonaName:           new(core.GetName()),
		PersonaState:          &core.PersonaState,
		PersonaStateFlags:     &core.PersonaStateFlags,
		PrimaryClanId:         &core.PrimaryClanID,
		ProfileState:          &core.ProfileState,
		ProfileUrl:            &core.ProfileURL,
		RealName:              &core.RealName,
		TimeCreated:           timestamppb.New(core.GetTimeCreated()),
		VisibilityState:       new(v1.VisibilityState(core.VisibilityState)),
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
			SteamId:      new(sid.Int64()),
		}
	}

	return friends
}

func toSettings(settings Settings) *v1.Settings {
	return &v1.Settings{
		PersonSettingsId:     &settings.PersonSettingsID,
		SteamId:              new(settings.SteamID.Int64()),
		ForumSignature:       &settings.ForumSignature,
		ForumProfileMessages: &settings.ForumProfileMessages,
		StatsHidden:          &settings.StatsHidden,
		CenterProjectiles:    settings.CenterProjectiles,
		CreatedOn:            timestamppb.New(settings.CreatedOn),
		UpdatedOn:            timestamppb.New(settings.UpdatedOn),
	}
}
