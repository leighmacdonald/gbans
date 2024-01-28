package repository

import (
	"context"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/errs"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"time"
)

type steamGroupRepository struct {
	db store.Database
}

func (s *steamGroupRepository) DropBanGroup(ctx context.Context, banGroup *domain.BanGroup) error {
	banGroup.IsEnabled = false
	banGroup.Deleted = true

	return s.BanSteamGroup(ctx, banGroup)
}

func NewSteamGroupRepository(database store.Database) domain.BanGroupRepository {
	return &steamGroupRepository{db: database}
}

func (s *steamGroupRepository) BanSteamGroup(ctx context.Context, banGroup *domain.BanGroup) error {
	if banGroup.BanGroupID > 0 {
		return s.updateBanGroup(ctx, banGroup)
	}

	return s.insertBanGroup(ctx, banGroup)
}

func (s *steamGroupRepository) insertBanGroup(ctx context.Context, banGroup *domain.BanGroup) error {
	const query = `
	INSERT INTO ban_group (source_id, target_id, group_id, group_name, is_enabled, deleted, note,
	unban_reason_text, origin, created_on, updated_on, valid_until, appeal_state)
	VALUES ($1, $2, $3, $4, $5, $6,$7, $8, $9, $10, $11, $12, $13)
	RETURNING ban_group_id`

	return errs.DBErr(s.db.
		QueryRow(ctx, query, banGroup.SourceID.Int64(), banGroup.TargetID.Int64(), banGroup.GroupID.Int64(),
			banGroup.GroupName, banGroup.IsEnabled, banGroup.Deleted, banGroup.Note, banGroup.UnbanReasonText, banGroup.Origin,
			banGroup.CreatedOn, banGroup.UpdatedOn, banGroup.ValidUntil, banGroup.AppealState).
		Scan(&banGroup.BanGroupID))
}

func (s *steamGroupRepository) updateBanGroup(ctx context.Context, banGroup *domain.BanGroup) error {
	banGroup.UpdatedOn = time.Now()

	return errs.DBErr(s.db.ExecUpdateBuilder(ctx, s.db.
		Builder().
		Update("ban_group").
		Set("source_id", banGroup.SourceID.Int64()).
		Set("target_id", banGroup.TargetID.Int64()).
		Set("group_name", banGroup.GroupName).
		Set("is_enabled", banGroup.IsEnabled).
		Set("deleted", banGroup.Deleted).
		Set("note", banGroup.Note).
		Set("unban_reason_text", banGroup.UnbanReasonText).
		Set("origin", banGroup.Origin).
		Set("updated_on", banGroup.UpdatedOn).
		Set("group_id", banGroup.GroupID.Int64()).
		Set("valid_until", banGroup.ValidUntil).
		Set("appeal_state", banGroup.AppealState).
		Where(sq.Eq{"ban_group_id": banGroup.BanGroupID})))
}

func (s *steamGroupRepository) GetBanGroup(ctx context.Context, groupID steamid.GID, banGroup *domain.BanGroup) error {
	query := s.db.
		Builder().
		Select("ban_group_id", "source_id", "target_id", "group_name", "is_enabled", "deleted",
			"note", "unban_reason_text", "origin", "created_on", "updated_on", "valid_until", "appeal_state", "group_id").
		From("ban_group").
		Where(sq.And{sq.Eq{"group_id": groupID.Int64()}, sq.Eq{"deleted": false}})

	var (
		sourceID   int64
		targetID   int64
		newGroupID int64
	)

	row, errQuery := s.db.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return errs.DBErr(errQuery)
	}

	if errScan := row.Scan(&banGroup.BanGroupID, &sourceID, &targetID, &banGroup.GroupName, &banGroup.IsEnabled,
		&banGroup.Deleted, &banGroup.Note, &banGroup.UnbanReasonText, &banGroup.Origin, &banGroup.CreatedOn,
		&banGroup.UpdatedOn, &banGroup.ValidUntil, &banGroup.AppealState, &newGroupID); errScan != nil {
		return errs.DBErr(errScan)
	}

	banGroup.SourceID = steamid.New(sourceID)
	banGroup.TargetID = steamid.New(targetID)
	banGroup.GroupID = steamid.NewGID(newGroupID)

	return nil
}

func (s *steamGroupRepository) GetBanGroupByID(ctx context.Context, banGroupID int64, banGroup *domain.BanGroup) error {
	query := s.db.
		Builder().
		Select("ban_group_id", "source_id", "target_id", "group_name", "is_enabled", "deleted",
			"note", "unban_reason_text", "origin", "created_on", "updated_on", "valid_until", "appeal_state", "group_id").
		From("ban_group").
		Where(sq.And{sq.Eq{"ban_group_id": banGroupID}, sq.Eq{"is_enabled": true}, sq.Eq{"deleted": false}})

	var (
		groupID  int64
		sourceID int64
		targetID int64
	)

	row, errQuery := s.db.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return errs.DBErr(errQuery)
	}

	if errScan := row.Scan(
		&banGroup.BanGroupID,
		&sourceID,
		&targetID,
		&banGroup.GroupName,
		&banGroup.IsEnabled,
		&banGroup.Deleted,
		&banGroup.Note,
		&banGroup.UnbanReasonText,
		&banGroup.Origin,
		&banGroup.CreatedOn,
		&banGroup.UpdatedOn,
		&banGroup.ValidUntil,
		&banGroup.AppealState,
		&groupID); errScan != nil {
		return errs.DBErr(errScan)
	}

	banGroup.SourceID = steamid.New(sourceID)
	banGroup.TargetID = steamid.New(targetID)
	banGroup.GroupID = steamid.NewGID(groupID)

	return nil
}

func (s *steamGroupRepository) GetBanGroups(ctx context.Context, filter domain.GroupBansQueryFilter) ([]domain.BannedGroupPerson, int64, error) {
	builder := s.db.
		Builder().
		Select("s.ban_group_id", "s.source_id", "s.target_id", "s.group_name", "s.is_enabled", "s.deleted",
			"s.note", "s.unban_reason_text", "s.origin", "s.created_on", "s.updated_on", "s.valid_until",
			"s.appeal_state", "s.group_id",
			"s.personaname as source_personaname", "s.avatarhash",
			"t.personaname as target_personaname", "t.avatarhash", "t.community_banned", "t.vac_bans", "t.game_bans").
		From("ban_group s").
		LeftJoin("person s ON s.steam_id = s.source_id").
		LeftJoin("person t ON t.steam_id = s.target_id")

	var constraints sq.And

	if !filter.Deleted {
		constraints = append(constraints, sq.Eq{"s.deleted": false})
	}

	if filter.Reason > 0 {
		constraints = append(constraints, sq.Eq{"s.reason": filter.Reason})
	}

	if filter.PermanentOnly {
		constraints = append(constraints, sq.Gt{"s.valid_until": time.Now()})
	}

	if filter.GroupID != "" {
		gid := steamid.NewGID(filter.GroupID)
		if !gid.Valid() {
			return nil, 0, steamid.ErrInvalidGID
		}

		constraints = append(constraints, sq.Eq{"s.group_id": gid.Int64()})
	}

	if filter.TargetID != "" {
		targetID, errTargetID := filter.TargetID.SID64(ctx)
		if errTargetID != nil {
			return nil, 0, errors.Join(errTargetID, errs.ErrTargetID)
		}

		constraints = append(constraints, sq.Eq{"s.target_id": targetID.Int64()})
	}

	if filter.SourceID != "" {
		sourceID, errSourceID := filter.SourceID.SID64(ctx)
		if errSourceID != nil {
			return nil, 0, errors.Join(errSourceID, errs.ErrSourceID)
		}

		constraints = append(constraints, sq.Eq{"s.source_id": sourceID.Int64()})
	}

	builder = filter.QueryFilter.ApplySafeOrder(builder, map[string][]string{
		"b.": {
			"ban_group_id", "source_id", "target_id", "group_name", "is_enabled", "deleted",
			"origin", "created_on", "updated_on", "valid_until", "appeal_state", "group_id",
		},
		"s.": {"source_personaname"},
		"t.": {"target_personaname", "community_banned", "vac_bans", "game_bans"},
	}, "ban_group_id")

	builder = filter.ApplyLimitOffsetDefault(builder).Where(constraints)

	rows, errRows := s.db.QueryBuilder(ctx, builder)
	if errRows != nil {
		if errors.Is(errRows, errs.ErrNoResult) {
			return []domain.BannedGroupPerson{}, 0, nil
		}

		return nil, 0, errs.DBErr(errRows)
	}

	defer rows.Close()

	var groups []domain.BannedGroupPerson

	for rows.Next() {
		var (
			group    domain.BannedGroupPerson
			groupID  int64
			sourceID int64
			targetID int64
		)

		if errScan := rows.Scan(
			&group.BanGroupID,
			&sourceID,
			&targetID,
			&group.GroupName,
			&group.IsEnabled,
			&group.Deleted,
			&group.Note,
			&group.UnbanReasonText,
			&group.Origin,
			&group.CreatedOn,
			&group.UpdatedOn,
			&group.ValidUntil,
			&group.AppealState,
			&groupID,
			&group.SourceTarget.SourcePersonaname, &group.SourceTarget.SourceAvatarhash,
			&group.SourceTarget.TargetPersonaname, &group.SourceTarget.TargetAvatarhash,
			&group.CommunityBanned, &group.VacBans, &group.GameBans,
		); errScan != nil {
			return nil, 0, errs.DBErr(errScan)
		}

		group.SourceID = steamid.New(sourceID)
		group.TargetID = steamid.New(targetID)
		group.GroupID = steamid.NewGID(groupID)

		groups = append(groups, group)
	}

	count, errCount := s.db.GetCount(ctx, s.db.
		Builder().
		Select("s.ban_group_id").
		From("ban_group s").
		Where(constraints))
	if errCount != nil {
		if errors.Is(errCount, errs.ErrNoResult) {
			return []domain.BannedGroupPerson{}, 0, nil
		}

		return nil, 0, errs.DBErr(errCount)
	}

	if groups == nil {
		groups = []domain.BannedGroupPerson{}
	}

	return groups, count, nil
}

func (s *steamGroupRepository) GetMembersList(ctx context.Context, parentID int64, list *domain.MembersList) error {
	row, err := s.db.QueryRowBuilder(ctx, s.db.
		Builder().
		Select("members_id", "parent_id", "members", "created_on", "updated_on").
		From("members").
		Where(sq.Eq{"parent_id": parentID}))
	if err != nil {
		return errs.DBErr(err)
	}

	return errs.DBErr(row.Scan(&list.MembersID, &list.ParentID, &list.Members, &list.CreatedOn, &list.UpdatedOn))
}

func (s *steamGroupRepository) SaveMembersList(ctx context.Context, list *domain.MembersList) error {
	if list.MembersID > 0 {
		list.UpdatedOn = time.Now()

		const update = `UPDATE members SET members = $2::jsonb, updated_on = $3 WHERE members_id = $1`

		return errs.DBErr(s.db.Exec(ctx, update, list.MembersID, list.Members, list.UpdatedOn))
	} else {
		const insert = `INSERT INTO members (parent_id, members, created_on, updated_on) 
		VALUES ($1, $2::jsonb, $3, $4) RETURNING members_id`

		return errs.DBErr(s.db.QueryRow(ctx, insert, list.ParentID, list.Members, list.CreatedOn, list.UpdatedOn).Scan(&list.MembersID))
	}
}
