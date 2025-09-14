package playerqueue

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

func NewPlayerqueueRepository(db database.Database, persons domain.PersonProvider) PlayerqueueRepository {
	return PlayerqueueRepository{db: db, persons: persons}
}

type PlayerqueueRepository struct {
	db      database.Database
	persons domain.PersonProvider
}

func (r PlayerqueueRepository) Message(ctx context.Context, messageID int64) (ChatLog, error) {
	row, err := r.db.QueryRowBuilder(ctx, nil, r.db.Builder().
		Select("m.message_id", "m.steam_id", "m.created_on", "m.personaname", "m.avatarhash", "p.permission_level", "m.body_md").
		From("playerqueue_messages m").
		LeftJoin("person p USING(steam_id)").
		Where(sq.And{sq.Eq{"m.deleted": false}, sq.Eq{"m.message_id": messageID}}))
	if err != nil {
		return ChatLog{}, r.db.DBErr(err)
	}

	var message ChatLog

	if errScan := row.Scan(&message.MessageID, &message.SteamID, &message.CreatedOn, &message.Personaname,
		&message.Avatarhash, &message.PermissionLevel, &message.BodyMD); errScan != nil {
		return ChatLog{}, r.db.DBErr(errScan)
	}

	return message, nil
}

func (r PlayerqueueRepository) Delete(ctx context.Context, messageID ...int64) error {
	return r.db.DBErr(r.db.ExecUpdateBuilder(ctx, nil, r.db.Builder().
		Update("playerqueue_messages").
		Set("deleted", true).
		Where(sq.Eq{"message_id": messageID})))
}

func (r PlayerqueueRepository) Save(ctx context.Context, message ChatLog) (ChatLog, error) {
	// Ensure player exists
	_, errPlayer := r.persons.GetOrCreatePersonBySteamID(ctx, nil, steamid.New(message.SteamID))
	if errPlayer != nil {
		return ChatLog{}, errPlayer
	}

	query, args, errQuery := r.db.Builder().
		Insert("playerqueue_messages").
		SetMap(map[string]interface{}{
			"steam_id":    message.SteamID,
			"created_on":  message.CreatedOn,
			"personaname": message.Personaname,
			"avatarhash":  message.Avatarhash,
			"body_md":     message.BodyMD,
		}).
		Suffix("RETURNING message_id").
		ToSql()
	if errQuery != nil {
		return ChatLog{}, r.db.DBErr(errQuery)
	}

	if err := r.db.QueryRow(ctx, nil, query, args...).Scan(&message.MessageID); err != nil {
		return message, r.db.DBErr(err)
	}

	return message, nil
}

func (r PlayerqueueRepository) Query(ctx context.Context, query PlayerqueueQueryOpts) ([]ChatLog, error) {
	builder := r.db.Builder().
		Select("m.message_id", "m.steam_id", "m.created_on", "m.personaname", "m.avatarhash",
			"p.permission_level", "m.body_md").
		From("playerqueue_messages m").
		LeftJoin("person p USING(steam_id)")

	if !query.Deleted {
		builder = builder.Where(sq.Eq{"m.deleted": false})
	}

	builder = query.ApplyLimitOffsetDefault(builder)
	builder = query.ApplySafeOrder(builder, map[string][]string{
		"m.": {
			"message_id", "steam_id", "created_on", "personaname", "avatarhash", "body_md", "deleted",
		},
	}, "steam_id")

	var msgs []ChatLog

	rows, errRows := r.db.QueryBuilder(ctx, nil, builder)
	if errRows != nil {
		return nil, r.db.DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var msg ChatLog
		if errScan := rows.Scan(&msg.MessageID, &msg.SteamID, &msg.CreatedOn, &msg.Personaname,
			&msg.Avatarhash, &msg.PermissionLevel, &msg.BodyMD); errScan != nil {
			return nil, r.db.DBErr(errScan)
		}

		msgs = append(msgs, msg)
	}

	return msgs, nil
}
