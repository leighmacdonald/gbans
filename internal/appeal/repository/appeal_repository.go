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

type appealRepository struct {
	store.Database
}

func NewAppealRepository(database store.Database) domain.AppealRepository {
	return &appealRepository{Database: database}
}

func (s *appealRepository) SaveBanMessage(ctx context.Context, message *domain.BanAppealMessage) error {
	var err error
	if message.BanMessageID > 0 {
		err = s.updateBanMessage(ctx, message)
	} else {
		err = s.insertBanMessage(ctx, message)
	}

	return err
}

func (s *appealRepository) updateBanMessage(ctx context.Context, message *domain.BanAppealMessage) error {
	message.UpdatedOn = time.Now()

	query := s.
		Builder().
		Update("ban_appeal").
		Set("deleted", message.Deleted).
		Set("author_id", message.AuthorID.Int64()).
		Set("updated_on", message.UpdatedOn).
		Set("message_md", message.MessageMD).
		Where(sq.Eq{"ban_message_id": message.BanMessageID})

	if errQuery := s.ExecUpdateBuilder(ctx, query); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return nil
}

func (s *appealRepository) insertBanMessage(ctx context.Context, message *domain.BanAppealMessage) error {
	const query = `
	INSERT INTO ban_appeal (
		ban_id, author_id, message_md, deleted, created_on, updated_on
	)
	VALUES ($1, $2, $3, $4, $5, $6)
	RETURNING ban_message_id
	`

	if errQuery := s.QueryRow(ctx, query,
		message.BanID,
		message.AuthorID.Int64(),
		message.MessageMD,
		message.Deleted,
		message.CreatedOn,
		message.UpdatedOn,
	).Scan(&message.BanMessageID); errQuery != nil {
		return errs.DBErr(errQuery)
	}

	return nil
}

func (s *appealRepository) GetBanMessages(ctx context.Context, banID int64) ([]domain.BanAppealMessage, error) {
	query := s.
		Builder().
		Select("a.ban_message_id", "a.ban_id", "a.author_id", "a.message_md", "a.deleted",
			"a.created_on", "a.updated_on", "p.avatarhash", "p.personaname", "p.permission_level").
		From("ban_appeal a").
		LeftJoin("person p ON a.author_id = p.steam_id").
		Where(sq.And{sq.Eq{"a.deleted": false}, sq.Eq{"a.ban_id": banID}}).
		OrderBy("a.created_on")

	rows, errQuery := s.QueryBuilder(ctx, query)
	if errQuery != nil {
		if errors.Is(errs.DBErr(errQuery), errs.ErrNoResult) {
			return nil, nil
		}
	}

	defer rows.Close()

	var messages []domain.BanAppealMessage

	for rows.Next() {
		var (
			msg      domain.BanAppealMessage
			authorID int64
		)

		if errScan := rows.Scan(
			&msg.BanMessageID,
			&msg.BanID,
			&authorID,
			&msg.MessageMD,
			&msg.Deleted,
			&msg.CreatedOn,
			&msg.UpdatedOn,
			&msg.Avatarhash,
			&msg.Personaname,
			&msg.PermissionLevel,
		); errScan != nil {
			return nil, errs.DBErr(errQuery)
		}

		msg.AuthorID = steamid.New(authorID)

		messages = append(messages, msg)
	}

	if messages == nil {
		return []domain.BanAppealMessage{}, nil
	}

	return messages, nil
}

func (s *appealRepository) GetBanMessageByID(ctx context.Context, banMessageID int, message *domain.BanAppealMessage) error {
	query := s.
		Builder().
		Select("a.ban_message_id", "a.ban_id", "a.author_id", "a.message_md", "a.deleted", "a.created_on",
			"a.updated_on", "p.avatarhash", "p.personaname", "p.permission_level").
		From("ban_appeal a").
		LeftJoin("person p ON a.author_id = p.steam_id").
		Where(sq.Eq{"a.ban_message_id": banMessageID})

	var authorID int64

	row, errQuery := s.QueryRowBuilder(ctx, query)
	if errQuery != nil {
		return errs.DBErr(errQuery)
	}

	if errScan := row.Scan(
		&message.BanMessageID,
		&message.BanID,
		&authorID,
		&message.MessageMD,
		&message.Deleted,
		&message.CreatedOn,
		&message.UpdatedOn,
		&message.Avatarhash,
		&message.Personaname,
		&message.PermissionLevel,
	); errScan != nil {
		return errs.DBErr(errScan)
	}

	message.AuthorID = steamid.New(authorID)

	return nil
}

func (s *appealRepository) DropBanMessage(ctx context.Context, message *domain.BanAppealMessage) error {
	query := s.
		Builder().
		Update("ban_appeal").
		Set("deleted", true).
		Where(sq.Eq{"ban_message_id": message.BanMessageID})

	if errExec := s.ExecUpdateBuilder(ctx, query); errExec != nil {
		return errs.DBErr(errExec)
	}

	message.Deleted = true

	return nil
}
