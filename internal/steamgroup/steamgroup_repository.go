package steamgroup

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type steamGroupRepository struct {
	db database.Database
}

func NewSteamGroupRepository(database database.Database) domain.BanGroupRepository {
	return &steamGroupRepository{db: database}
}

func (r *steamGroupRepository) TruncateCache(ctx context.Context) error {
	return r.db.DBErr(r.db.ExecDeleteBuilder(ctx, r.db.Builder().Delete("steam_group_members")))
}

func (r *steamGroupRepository) InsertCache(ctx context.Context, groupID steamid.SteamID, entries []int64) error {
	const query = "INSERT INTO steam_group_members (steam_id, group_id, created_on) VALUES ($1, $2, $3)"

	batch := pgx.Batch{}
	now := time.Now()

	for _, entrySteamID := range entries {
		batch.Queue(query, entrySteamID, groupID.Int64(), now)
	}

	batchResults := r.db.SendBatch(ctx, &batch)
	if errCloseBatch := batchResults.Close(); errCloseBatch != nil {
		return errors.Join(errCloseBatch, domain.ErrCloseBatch)
	}

	return nil
}

func (r *steamGroupRepository) Delete(ctx context.Context, banGroup *domain.BanGroup) error {
	banGroup.IsEnabled = false
	banGroup.Deleted = true

	return r.Ban(ctx, banGroup)
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

func (r *steamGroupRepository) GetByID(ctx context.Context, banGroupID int64) (domain.BannedGroupPerson, error) {
	query := r.db.
		Builder().
		Select("b.ban_group_id", "b.source_id", "b.target_id", "b.group_name", "b.is_enabled", "b.deleted",
			"b.note", "b.unban_reason_text", "b.origin", "b.created_on", "b.updated_on", "b.valid_until", "b.appeal_state", "b.group_id",
			"s.personaname", "s.avatarhash", "t.personaname", "t.avatarhash").
		From("ban_group b").
		LeftJoin("person s ON b.source_id = s.steam_id").
		LeftJoin("person t ON b.target_id = t.steam_id").
		Where(sq.And{sq.Eq{"b.ban_group_id": banGroupID}, sq.Eq{"b.is_enabled": true}, sq.Eq{"b.deleted": false}})

	var (
		banGroup domain.BannedGroupPerson
		groupID  int64
		sourceID int64
		targetID int64
	)

	row, errQuery := r.db.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return banGroup, r.db.DBErr(errQuery)
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
		&groupID,
		&banGroup.SourcePersonaname,
		&banGroup.SourceAvatarhash,
		&banGroup.TargetPersonaname,
		&banGroup.TargetAvatarhash); errScan != nil {
		return banGroup, r.db.DBErr(errScan)
	}

	banGroup.SourceID = steamid.New(sourceID)
	banGroup.TargetID = steamid.New(targetID)
	banGroup.GroupID = steamid.New(groupID)

	return banGroup, nil
}

func (r *steamGroupRepository) Get(ctx context.Context, filter domain.GroupBansQueryFilter) ([]domain.BannedGroupPerson, error) {
	builder := r.db.
		Builder().
		Select("g.ban_group_id", "g.source_id", "g.target_id", "g.group_name", "g.is_enabled", "g.deleted",
			"g.note", "g.unban_reason_text", "g.origin", "g.created_on", "g.updated_on", "g.valid_until",
			"g.appeal_state", "g.group_id",
			"s.personaname as source_personaname", "s.avatarhash",
			"coalesce(t.personaname, '') as target_personaname", "coalesce(t.avatarhash, '')",
			"coalesce(t.community_banned, false)", "coalesce(t.vac_bans, 0)", "coalesce(t.game_bans, 0)").
		From("ban_group g").
		LeftJoin("person s ON s.steam_id = g.source_id").
		LeftJoin("person t ON t.steam_id = g.target_id")

	var constraints sq.And

	if !filter.Deleted {
		constraints = append(constraints, sq.Eq{"g.deleted": false})
	}

	builder = builder.Where(constraints)

	rows, errRows := r.db.QueryBuilder(ctx, builder)
	if errRows != nil {
		if errors.Is(errRows, domain.ErrNoResult) {
			return []domain.BannedGroupPerson{}, nil
		}

		return nil, r.db.DBErr(errRows)
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
			return nil, r.db.DBErr(errScan)
		}

		group.SourceID = steamid.New(sourceID)
		group.TargetID = steamid.New(targetID)
		group.GroupID = steamid.New(groupID)

		groups = append(groups, group)
	}

	if groups == nil {
		groups = []domain.BannedGroupPerson{}
	}

	return groups, nil
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
	}

	const insert = `INSERT INTO members (parent_id, members, created_on, updated_on) 
		VALUES ($1, $2::jsonb, $3, $4) RETURNING members_id`

	return r.db.DBErr(r.db.QueryRow(ctx, insert, list.ParentID, list.Members, list.CreatedOn, list.UpdatedOn).Scan(&list.MembersID))
}
