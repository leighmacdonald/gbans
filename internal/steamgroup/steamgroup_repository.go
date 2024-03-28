package steamgroup

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type steamGroupRepository struct {
	db database.Database
}

func (r *steamGroupRepository) Delete(ctx context.Context, banGroup *domain.BanGroup) error {
	banGroup.IsEnabled = false
	banGroup.Deleted = true

	return r.Ban(ctx, banGroup)
}

func NewSteamGroupRepository(database database.Database) domain.BanGroupRepository {
	return &steamGroupRepository{db: database}
}

func (r *steamGroupRepository) Save(ctx context.Context, banGroup *domain.BanGroup) error {
	if banGroup.BanGroupID > 0 {
		return r.updateBanGroup(ctx, banGroup)
	}

	return r.insertBanGroup(ctx, banGroup)
}

func (r *steamGroupRepository) Ban(ctx context.Context, banGroup *domain.BanGroup) error {
	if banGroup.BanGroupID > 0 {
		return r.updateBanGroup(ctx, banGroup)
	}

	return r.insertBanGroup(ctx, banGroup)
}

func (r *steamGroupRepository) insertBanGroup(ctx context.Context, banGroup *domain.BanGroup) error {
	const query = `
	INSERT INTO ban_group (source_id, target_id, group_id, group_name, is_enabled, deleted, note,
	unban_reason_text, origin, created_on, updated_on, valid_until, appeal_state)
	VALUES ($1, $2, $3, $4, $5, $6,$7, $8, $9, $10, $11, $12, $13)
	RETURNING ban_group_id`

	return r.db.DBErr(r.db.
		QueryRow(ctx, query, banGroup.SourceID.Int64(), banGroup.TargetID.Int64(), banGroup.GroupID.Int64(),
			banGroup.GroupName, banGroup.IsEnabled, banGroup.Deleted, banGroup.Note, banGroup.UnbanReasonText, banGroup.Origin,
			banGroup.CreatedOn, banGroup.UpdatedOn, banGroup.ValidUntil, banGroup.AppealState).
		Scan(&banGroup.BanGroupID))
}

func (r *steamGroupRepository) updateBanGroup(ctx context.Context, banGroup *domain.BanGroup) error {
	banGroup.UpdatedOn = time.Now()

	return r.db.DBErr(r.db.ExecUpdateBuilder(ctx, r.db.
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

func (r *steamGroupRepository) GetByGID(ctx context.Context, groupID steamid.SteamID, banGroup *domain.BanGroup) error {
	query := r.db.
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

	row, errQuery := r.db.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return r.db.DBErr(errQuery)
	}

	if errScan := row.Scan(&banGroup.BanGroupID, &sourceID, &targetID, &banGroup.GroupName, &banGroup.IsEnabled,
		&banGroup.Deleted, &banGroup.Note, &banGroup.UnbanReasonText, &banGroup.Origin, &banGroup.CreatedOn,
		&banGroup.UpdatedOn, &banGroup.ValidUntil, &banGroup.AppealState, &newGroupID); errScan != nil {
		return r.db.DBErr(errScan)
	}

	banGroup.SourceID = steamid.New(sourceID)
	banGroup.TargetID = steamid.New(targetID)
	banGroup.GroupID = steamid.New(newGroupID)

	return nil
}

func (r *steamGroupRepository) GetByID(ctx context.Context, banGroupID int64, banGroup *domain.BanGroup) error {
	query := r.db.
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

	row, errQuery := r.db.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return r.db.DBErr(errQuery)
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
		return r.db.DBErr(errScan)
	}

	banGroup.SourceID = steamid.New(sourceID)
	banGroup.TargetID = steamid.New(targetID)
	banGroup.GroupID = steamid.New(groupID)

	return nil
}

func (r *steamGroupRepository) Get(ctx context.Context, filter domain.GroupBansQueryFilter) ([]domain.BannedGroupPerson, int64, error) {
	builder := r.db.
		Builder().
		Select("g.ban_group_id", "g.source_id", "g.target_id", "g.group_name", "g.is_enabled", "g.deleted",
			"g.note", "g.unban_reason_text", "g.origin", "g.created_on", "g.updated_on", "g.valid_until",
			"g.appeal_state", "g.group_id",
			"s.personaname as source_personaname", "s.avatarhash",
			"t.personaname as target_personaname", "t.avatarhash", "t.community_banned", "t.vac_bans", "t.game_bans").
		From("ban_group g").
		LeftJoin("person s ON s.steam_id = g.source_id").
		LeftJoin("person t ON t.steam_id = g.target_id")

	var constraints sq.And

	if !filter.Deleted {
		constraints = append(constraints, sq.Eq{"g.deleted": false})
	}

	if filter.Reason > 0 {
		constraints = append(constraints, sq.Eq{"g.reason": filter.Reason})
	}

	if filter.PermanentOnly {
		constraints = append(constraints, sq.Gt{"g.valid_until": time.Now()})
	}

	if filter.GroupID != "" {
		gid := steamid.New(filter.GroupID)
		if !gid.Valid() {
			return nil, 0, steamid.ErrInvalidGID
		}

		constraints = append(constraints, sq.Eq{"g.group_id": gid.Int64()})
	}

	if sid, ok := filter.TargetSteamID(ctx); ok {
		constraints = append(constraints, sq.Eq{"g.target_id": sid})
	}

	if sid, ok := filter.SourceSteamID(ctx); ok {
		constraints = append(constraints, sq.Eq{"g.source_id": sid})
	}

	builder = filter.QueryFilter.ApplySafeOrder(builder, map[string][]string{
		"g.": {
			"ban_group_id", "source_id", "target_id", "group_name", "is_enabled", "deleted",
			"origin", "created_on", "updated_on", "valid_until", "appeal_state", "group_id",
		},
		"s.": {"source_personaname"},
		"t.": {"target_personaname", "community_banned", "vac_bans", "game_bans"},
	}, "ban_group_id")

	builder = filter.ApplyLimitOffsetDefault(builder).Where(constraints)

	rows, errRows := r.db.QueryBuilder(ctx, builder)
	if errRows != nil {
		if errors.Is(errRows, domain.ErrNoResult) {
			return []domain.BannedGroupPerson{}, 0, nil
		}

		return nil, 0, r.db.DBErr(errRows)
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
			return nil, 0, r.db.DBErr(errScan)
		}

		group.SourceID = steamid.New(sourceID)
		group.TargetID = steamid.New(targetID)
		group.GroupID = steamid.New(groupID)

		groups = append(groups, group)
	}

	count, errCount := r.db.GetCount(ctx, r.db.
		Builder().
		Select("g.ban_group_id").
		From("ban_group g").
		Where(constraints))
	if errCount != nil {
		if errors.Is(errCount, domain.ErrNoResult) {
			return []domain.BannedGroupPerson{}, 0, nil
		}

		return nil, 0, r.db.DBErr(errCount)
	}

	if groups == nil {
		groups = []domain.BannedGroupPerson{}
	}

	return groups, count, nil
}

func (r *steamGroupRepository) GetMembersList(ctx context.Context, parentID int64, list *domain.MembersList) error {
	row, err := r.db.QueryRowBuilder(ctx, r.db.
		Builder().
		Select("members_id", "parent_id", "members", "created_on", "updated_on").
		From("members").
		Where(sq.Eq{"parent_id": parentID}))
	if err != nil {
		return r.db.DBErr(err)
	}

	return r.db.DBErr(row.Scan(&list.MembersID, &list.ParentID, &list.Members, &list.CreatedOn, &list.UpdatedOn))
}

func (r *steamGroupRepository) SaveMembersList(ctx context.Context, list *domain.MembersList) error {
	if list.MembersID > 0 {
		list.UpdatedOn = time.Now()

		const update = `UPDATE members SET members = $2::jsonb, updated_on = $3 WHERE members_id = $1`

		return r.db.DBErr(r.db.Exec(ctx, update, list.MembersID, list.Members, list.UpdatedOn))
	} else {
		const insert = `INSERT INTO members (parent_id, members, created_on, updated_on) 
		VALUES ($1, $2::jsonb, $3, $4) RETURNING members_id`

		return r.db.DBErr(r.db.QueryRow(ctx, insert, list.ParentID, list.Members, list.CreatedOn, list.UpdatedOn).Scan(&list.MembersID))
	}
}
