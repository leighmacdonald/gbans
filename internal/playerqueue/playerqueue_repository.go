package playerqueue

import (
	"context"

	"github.com/gofrs/uuid/v5"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

func NewPlayerqueueRepository(db database.Database, persons domain.PersonUsecase) domain.PlayerqueueRepository {
	return playerqueueRepository{db: db, persons: persons}
}

type playerqueueRepository struct {
	db      database.Database
	persons domain.PersonUsecase
}

func (r playerqueueRepository) Save(ctx context.Context, message domain.Message) (domain.Message, error) {
	uuidVal, errUUID := uuid.NewV4()
	if errUUID != nil {
		return domain.Message{}, errUUID //nolint:wrapcheck
	}

	message.MessageID = uuidVal

	// Ensure player exists
	_, errPlayer := r.persons.GetOrCreatePersonBySteamID(ctx, nil, steamid.New(message.SteamID))
	if errPlayer != nil {
		return domain.Message{}, errPlayer
	}

	if err := r.db.ExecInsertBuilder(ctx, nil, r.db.Builder().
		Insert("playerqueue_messages").
		SetMap(map[string]interface{}{
			"message_id":  message.MessageID,
			"steam_id":    message.SteamID,
			"created_on":  message.CreatedOn,
			"personaname": message.Personaname,
			"avatarhash":  message.Avatarhash,
			"body_md":     message.BodyMD,
		})); err != nil {
		return message, r.db.DBErr(err)
	}

	return message, nil
}

func (r playerqueueRepository) Query(ctx context.Context, query domain.PlayerqueueQueryOpts) ([]domain.Message, error) {
	builder := r.db.Builder().
		Select("m.message_id", "m.steam_id", "m.created_on", "m.personaname", "m.avatarhash", "p.permission_level", "m.body_md").
		From("playerqueue_messages m").
		LeftJoin("person p USING(steam_id)")

	builder = query.ApplyLimitOffsetDefault(builder)
	builder = query.ApplySafeOrder(builder, map[string][]string{
		"m.": {
			"message_id", "steam_id", "created_on", "personaname", "avatarhash", "body_md",
		},
	}, "steam_id")

	var msgs []domain.Message

	rows, errRows := r.db.QueryBuilder(ctx, nil, builder)
	if errRows != nil {
		return nil, r.db.DBErr(errRows)
	}

	defer rows.Close()

	for rows.Next() {
		var msg domain.Message
		if errScan := rows.Scan(&msg.MessageID, &msg.SteamID, &msg.CreatedOn, &msg.Personaname,
			&msg.Avatarhash, &msg.PermissionLevel, &msg.BodyMD); errScan != nil {
			return nil, r.db.DBErr(errScan)
		}

		msgs = append(msgs, msg)
	}

	return msgs, nil
}
