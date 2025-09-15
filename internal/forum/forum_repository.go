package forum

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

type ForumRepository struct {
	db database.Database
}

func NewForumRepository(database database.Database) ForumRepository {
	return ForumRepository{db: database}
}

func (f *ForumRepository) ForumCategories(ctx context.Context) ([]ForumCategory, error) {
	rows, errRows := f.db.QueryBuilder(ctx, nil, f.db.
		Builder().
		Select("forum_category_id", "title", "description", "ordering", "updated_on", "created_on").
		From("forum_category").
		OrderBy("ordering"))
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}

	defer rows.Close()

	var categories []ForumCategory

	for rows.Next() {
		var fc ForumCategory
		if errScan := rows.Scan(&fc.ForumCategoryID, &fc.Title, &fc.Description, &fc.Ordering, &fc.UpdatedOn, &fc.CreatedOn); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		categories = append(categories, fc)
	}

	return categories, nil
}

func (f *ForumRepository) ForumCategorySave(ctx context.Context, category *ForumCategory) error {
	category.UpdatedOn = time.Now()
	if category.ForumCategoryID > 0 {
		return database.DBErr(f.db.ExecUpdateBuilder(ctx, nil, f.db.
			Builder().
			Update("forum_category").
			SetMap(map[string]interface{}{
				"title":       category.Title,
				"description": category.Description,
				"ordering":    category.Ordering,
				"updated_on":  category.UpdatedOn,
			}).
			Where(sq.Eq{"forum_category_id": category.ForumCategoryID})))
	}

	category.CreatedOn = category.UpdatedOn

	return database.DBErr(f.db.ExecInsertBuilderWithReturnValue(ctx, nil, f.db.
		Builder().
		Insert("forum_category").
		SetMap(map[string]interface{}{
			"title":       category.Title,
			"description": category.Description,
			"ordering":    category.Ordering,
			"created_on":  category.CreatedOn,
			"updated_on":  category.UpdatedOn,
		}).
		Suffix("RETURNING forum_category_id"), &category.ForumCategoryID))
}

func (f *ForumRepository) ForumCategory(ctx context.Context, categoryID int, category *ForumCategory) error {
	row, errRow := f.db.QueryRowBuilder(ctx, nil, f.db.
		Builder().
		Select("forum_category_id", "title", "description", "ordering", "created_on", "created_on").
		From("forum_category").
		Where(sq.Eq{"forum_category_id": categoryID}))
	if errRow != nil {
		return database.DBErr(errRow)
	}

	return database.DBErr(row.Scan(&category.ForumCategoryID, &category.Title, &category.Description, &category.Ordering,
		&category.CreatedOn, &category.UpdatedOn))
}

func (f *ForumRepository) ForumCategoryDelete(ctx context.Context, categoryID int) error {
	return database.DBErr(f.db.ExecDeleteBuilder(ctx, nil, f.db.
		Builder().
		Delete("forum_category").
		Where(sq.Eq{"forum_category_id": categoryID})))
}

func (f *ForumRepository) Forums(ctx context.Context) ([]Forum, error) {
	fromSelect := f.db.
		Builder().
		Select("DISTINCT ON (s.forum_id) s.forum_id", "s.forum_category_id", "s.title", "s.description", "s.last_thread_id",
			"s.count_threads", "s.count_messages", "s.ordering", "s.created_on", "s.updated_on", "s.permission_level",
			"t.forum_thread_id", "t.source_id", "p.personaname", "p.avatarhash", "t.created_on", "t.title").
		From("forum s").
		LeftJoin("forum_thread t ON s.last_thread_id = t.forum_thread_id").
		LeftJoin("forum_message m ON t.forum_thread_id = m.forum_thread_id").
		LeftJoin("person p ON p.steam_id = m.source_id")

	rows, errRows := f.db.QueryBuilder(ctx, nil, f.db.
		Builder().
		Select("x.*").
		FromSelect(fromSelect, "x").
		OrderBy("x.ordering"))
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}

	defer rows.Close()

	var forums []Forum

	for rows.Next() {
		var (
			lastID           *int64
			frm              Forum
			lastForumTheadID *int64
			lastSourceID     *steamid.SteamID
			lastPersonaname  *string
			lastAvatarhash   *string
			lastCreatedOn    *time.Time
			lastTitle        *string
		)

		if errScan := rows.Scan(&frm.ForumID, &frm.ForumCategoryID, &frm.Title, &frm.Description,
			&lastID, &frm.CountThreads, &frm.CountMessages,
			&frm.Ordering, &frm.CreatedOn, &frm.UpdatedOn, &frm.PermissionLevel,
			&lastForumTheadID, &lastSourceID, &lastPersonaname, &lastAvatarhash,
			&lastCreatedOn, &lastTitle); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		if lastID != nil {
			frm.LastThreadID = *lastID
		}

		if lastForumTheadID != nil {
			frm.RecentForumThreadID = *lastForumTheadID
			frm.RecentSourceID = *lastSourceID
			frm.RecentPersonaname = *lastPersonaname
			frm.RecentAvatarhash = *lastAvatarhash
			frm.RecentCreatedOn = *lastCreatedOn
			frm.RecentForumTitle = *lastTitle
		}

		forums = append(forums, frm)
	}

	return forums, nil
}

func (f *ForumRepository) ForumSave(ctx context.Context, forum *Forum) error {
	forum.UpdatedOn = time.Now()

	var lastThreadID *int64

	if forum.LastThreadID > 0 {
		lastThreadID = &forum.LastThreadID
	}

	if forum.ForumID > 0 {
		return database.DBErr(f.db.ExecUpdateBuilder(ctx, nil, f.db.
			Builder().
			Update("forum").
			SetMap(map[string]interface{}{
				"forum_category_id": forum.ForumCategoryID,
				"title":             forum.Title,
				"description":       forum.Description,
				"last_thread_id":    lastThreadID,
				"count_threads":     forum.CountThreads,
				"count_messages":    forum.CountMessages,
				"ordering":          forum.Ordering,
				"permission_level":  forum.PermissionLevel,
				"updated_on":        forum.UpdatedOn,
			}).
			Where(sq.Eq{"forum_id": forum.ForumID})))
	}

	forum.CreatedOn = time.Now()

	return database.DBErr(f.db.ExecInsertBuilderWithReturnValue(ctx, nil, f.db.
		Builder().
		Insert("forum").
		SetMap(map[string]interface{}{
			"forum_category_id": forum.ForumCategoryID,
			"title":             forum.Title,
			"description":       forum.Description,
			"last_thread_id":    lastThreadID,
			"count_threads":     forum.CountThreads,
			"count_messages":    forum.CountMessages,
			"ordering":          forum.Ordering,
			"permission_level":  forum.PermissionLevel,
			"created_on":        forum.CreatedOn,
			"updated_on":        forum.UpdatedOn,
		}).
		Suffix("RETURNING forum_id"), &forum.ForumID))
}

func (f *ForumRepository) Forum(ctx context.Context, forumID int, forum *Forum) error {
	row, errRow := f.db.QueryRowBuilder(ctx, nil, f.db.
		Builder().
		Select("forum_id", "forum_category_id", "title", "description", "last_thread_id",
			"count_threads", "count_messages", "ordering", "created_on", "updated_on", "permission_level").
		From("forum").
		Where(sq.Eq{"forum_id": forumID}))
	if errRow != nil {
		return database.DBErr(errRow)
	}

	var lastThreadID *int64
	if err := row.Scan(&forum.ForumID, &forum.ForumCategoryID, &forum.Title, &forum.Description,
		&lastThreadID, &forum.CountThreads, &forum.CountMessages, &forum.Ordering,
		&forum.CreatedOn, &forum.UpdatedOn, &forum.PermissionLevel); err != nil {
		return database.DBErr(err)
	}

	if lastThreadID != nil {
		forum.LastThreadID = *lastThreadID
	}

	return nil
}

func (f *ForumRepository) ForumDelete(ctx context.Context, forumID int) error {
	return database.DBErr(f.db.ExecDeleteBuilder(ctx, nil, f.db.
		Builder().
		Delete("forum").
		Where(sq.Eq{"forum_id": forumID})))
}

func (f *ForumRepository) ForumThreadSave(ctx context.Context, thread *ForumThread) error {
	thread.UpdatedOn = time.Now()
	if thread.ForumThreadID > 0 {
		return database.DBErr(f.db.ExecUpdateBuilder(ctx, nil, f.db.
			Builder().
			Update("forum_thread").
			SetMap(map[string]interface{}{
				"forum_id":   thread.ForumID,
				"source_id":  thread.SourceID.Int64(),
				"title":      thread.Title,
				"sticky":     thread.Sticky,
				"locked":     thread.Locked,
				"views":      thread.Views,
				"updated_on": thread.UpdatedOn,
			}).
			Where(sq.Eq{"forum_thread_id": thread.ForumThreadID})))
	}

	thread.CreatedOn = time.Now()

	if errInsert := f.db.ExecInsertBuilderWithReturnValue(ctx, nil, f.db.
		Builder().
		Insert("forum_thread").
		SetMap(map[string]interface{}{
			"forum_id":   thread.ForumID,
			"source_id":  thread.SourceID.Int64(),
			"title":      thread.Title,
			"sticky":     thread.Sticky,
			"locked":     thread.Locked,
			"views":      thread.Views,
			"created_on": thread.CreatedOn,
			"updated_on": thread.UpdatedOn,
		}).
		Suffix("RETURNING forum_thread_id"), &thread.ForumThreadID); errInsert != nil {
		return database.DBErr(errInsert)
	}

	return database.DBErr(f.db.ExecUpdateBuilder(ctx, nil, f.db.
		Builder().
		Update("forum").
		Set("count_threads", sq.Expr("count_threads+1")).
		Set("last_thread_id", thread.ForumThreadID).
		Where(sq.Eq{"forum_id": thread.ForumID})))
}

func (f *ForumRepository) ForumThread(ctx context.Context, forumThreadID int64, thread *ForumThread) error {
	row, errRow := f.db.QueryRowBuilder(ctx, nil, f.db.
		Builder().
		Select("t.forum_thread_id", "t.forum_id", "t.source_id", "t.title", "t.sticky",
			"t.locked", "t.views", "t.created_on", "t.updated_on", "p.personaname", "p.avatarhash",
			"p.permission_level").
		From("forum_thread t").
		LeftJoin("person p ON p.steam_id = t.source_id").
		Where(sq.Eq{"t.forum_thread_id": forumThreadID}))
	if errRow != nil {
		return database.DBErr(errRow)
	}

	return database.DBErr(row.Scan(&thread.ForumThreadID, &thread.ForumID, &thread.SourceID, &thread.Title,
		&thread.Sticky, &thread.Locked, &thread.Views, &thread.CreatedOn,
		&thread.UpdatedOn, &thread.Personaname, &thread.Avatarhash, &thread.PermissionLevel))
}

func (f *ForumRepository) ForumThreadIncrView(ctx context.Context, forumThreadID int64) error {
	return database.DBErr(f.db.ExecUpdateBuilder(ctx, nil, f.db.
		Builder().
		Update("forum_thread").
		Set("views", sq.Expr("views+1")).
		Where(sq.Eq{"forum_thread_id": forumThreadID})))
}

func (f *ForumRepository) ForumThreadDelete(ctx context.Context, forumThreadID int64) error {
	return database.DBErr(f.db.ExecDeleteBuilder(ctx, nil, f.db.
		Builder().
		Delete("forum_thread").
		Where(sq.Eq{"forum_thread_id": forumThreadID})))
}

func (f *ForumRepository) ForumThreads(ctx context.Context, filter ThreadQueryFilter) ([]ThreadWithSource, error) {
	if filter.ForumID <= 0 {
		return nil, domain.ErrInvalidThread
	}

	// todo deleted archive
	constraints := sq.And{sq.Eq{"forum_id": filter.ForumID}}

	builder := f.db.
		Builder().
		Select("t.forum_thread_id", "t.forum_id", "t.source_id", "t.title", "t.sticky",
			"t.locked", "t.views", "t.created_on", "t.updated_on", "p.personaname", "p.avatarhash", "p.permission_level",
			"a.steam_id", "a.personaname", "a.avatarhash", "a.forum_message_id", "a.created_on", "c.message_count").
		From("forum_thread t").
		LeftJoin("person p ON p.steam_id = t.source_id").
		InnerJoin(`
			LATERAL (SELECT m.*, p2.personaname, p2.avatarhash, p2.steam_id
					 FROM forum_message m
					 LEFT JOIN public.person p2 on m.source_id = p2.steam_id
                     WHERE m.forum_thread_id = t.forum_thread_id
			         ORDER BY m.forum_message_id DESC
			         LIMIT 1) a ON TRUE`).
		InnerJoin(`
			LATERAL (SELECT count(m.forum_message_id) as message_count
                     FROM forum_message m
                     WHERE m.forum_thread_id = t.forum_thread_id
					) c ON TRUE`).
		OrderBy("t.sticky DESC, a.updated_on DESC").Where(constraints)

	rows, errRows := f.db.QueryBuilder(ctx, nil, builder)
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}

	defer rows.Close()

	var threads []ThreadWithSource

	for rows.Next() {
		var (
			tws                  ThreadWithSource
			RecentForumMessageID *int64
			RecentCreatedOn      *time.Time
			RecentSteamID        *string
			RecentPersonaname    *string
			RecentAvatarHash     *string
		)

		if errScan := rows.
			Scan(&tws.ForumThreadID, &tws.ForumID, &tws.SourceID, &tws.Title, &tws.Sticky,
				&tws.Locked, &tws.Views, &tws.CreatedOn, &tws.UpdatedOn, &tws.Personaname, &tws.Avatarhash,
				&tws.PermissionLevel, &RecentSteamID, &RecentPersonaname, &RecentAvatarHash, &RecentForumMessageID,
				&RecentCreatedOn, &tws.Replies); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		if RecentForumMessageID != nil {
			tws.RecentForumMessageID = *RecentForumMessageID
			tws.RecentCreatedOn = *RecentCreatedOn
			tws.RecentSteamID = *RecentSteamID
			tws.RecentPersonaname = *RecentPersonaname
			tws.RecentAvatarhash = *RecentAvatarHash
		}

		threads = append(threads, tws)
	}

	return threads, nil
}

func (f *ForumRepository) ForumIncrMessageCount(ctx context.Context, forumID int, incr bool) error {
	builder := f.db.
		Builder().
		Update("forum").
		Where(sq.Eq{"forum_id": forumID})

	if incr {
		builder = builder.Set("count_messages", sq.Expr("count_messages+1"))
	} else {
		builder = builder.Set("count_messages", sq.Expr("count_messages-1"))
	}

	return database.DBErr(f.db.ExecUpdateBuilder(ctx, nil, builder))
}

func (f *ForumRepository) ForumMessageSave(ctx context.Context, message *ForumMessage) error {
	message.UpdatedOn = time.Now()
	if message.ForumMessageID > 0 {
		return database.DBErr(f.db.ExecUpdateBuilder(ctx, nil, f.db.
			Builder().
			Update("forum_message").
			SetMap(map[string]interface{}{
				"forum_thread_id": message.ForumThreadID,
				"source_id":       message.SourceID.Int64(),
				"body_md":         message.BodyMD,
				"updated_on":      message.UpdatedOn,
			}).
			Where(sq.Eq{"forum_message_id": message.ForumMessageID})))
	}

	message.CreatedOn = time.Now()

	if errInsert := f.db.ExecInsertBuilderWithReturnValue(ctx, nil, f.db.
		Builder().
		Insert("forum_message").
		SetMap(map[string]interface{}{
			"forum_thread_id": message.ForumThreadID,
			"source_id":       message.SourceID.Int64(),
			"body_md":         message.BodyMD,
			"created_on":      message.CreatedOn,
			"updated_on":      message.UpdatedOn,
		}).
		Suffix("RETURNING forum_message_id"), &message.ForumMessageID); errInsert != nil {
		return database.DBErr(errInsert)
	}

	return database.DBErr(f.db.ExecUpdateBuilder(ctx, nil, f.db.
		Builder().
		Update("forum_thread").
		Set("updated_on", message.CreatedOn).
		Where(sq.Eq{"forum_thread_id": message.ForumThreadID})))
}

func (f *ForumRepository) ForumRecentActivity(ctx context.Context, limit uint64, permissionLevel permission.Privilege) ([]ForumMessage, error) {
	expr, _, errExpr := sq.Expr(`
			LATERAL (
				SELECT m.forum_message_id,
					   m.forum_thread_id,
					   m.created_on,
					   m.updated_on,
					   p.steam_id,
					   p.personaname,
					   p.avatarhash,
					   p.permission_level
				FROM forum_message m
				LEFT JOIN person p on m.source_id = p.steam_id
				WHERE m.forum_thread_id = t.forum_thread_id AND $1 >= s.permission_level
				ORDER BY m.forum_message_id DESC
				LIMIT 1
				) m on TRUE`, permissionLevel).ToSql()
	if errExpr != nil {
		return nil, database.DBErr(errExpr)
	}

	builder := f.db.
		Builder().
		Select("t.forum_thread_id", "m.forum_message_id",
			"m.steam_id", "m.created_on", "m.updated_on", "m.personaname",
			"m.avatarhash", "s.permission_level", "t.title").
		From("forum_thread t").
		LeftJoin("forum s ON s.forum_id = t.forum_id").
		InnerJoin(expr).
		Where(sq.GtOrEq{"m.permission_level": permissionLevel}).
		OrderBy("t.updated_on DESC").
		Limit(limit)

	rows, errRows := f.db.QueryBuilder(ctx, nil, builder)
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}

	defer rows.Close()

	var messages []ForumMessage

	for rows.Next() {
		var msg ForumMessage
		if errScan := rows.Scan(&msg.ForumThreadID, &msg.ForumMessageID, &msg.SourceID,
			&msg.CreatedOn, &msg.UpdatedOn, &msg.Personaname,
			&msg.Avatarhash, &msg.PermissionLevel, &msg.Title); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func (f *ForumRepository) ForumMessage(ctx context.Context, messageID int64, forumMessage *ForumMessage) error {
	row, errRow := f.db.QueryRowBuilder(ctx, nil, f.db.
		Builder().
		Select("m.forum_message_id", "m.forum_thread_id", "m.source_id", "m.body_md", "m.created_on", "m.updated_on",
			"p.personaname", "p.avatarhash", "p.permission_level", "coalesce(s.forum_signature, '')").
		From("forum_message m").
		LeftJoin("person p ON p.steam_id = m.source_id").
		LeftJoin("person_settings s ON s.steam_id = m.source_id").
		Where(sq.Eq{"forum_message_id": messageID}))
	if errRow != nil {
		return database.DBErr(errRow)
	}

	return database.DBErr(row.Scan(&forumMessage.ForumMessageID, &forumMessage.ForumThreadID, &forumMessage.SourceID,
		&forumMessage.BodyMD, &forumMessage.CreatedOn, &forumMessage.UpdatedOn, &forumMessage.Personaname,
		&forumMessage.Avatarhash, &forumMessage.PermissionLevel, &forumMessage.Signature))
}

func (f *ForumRepository) ForumMessages(ctx context.Context, filters ThreadMessagesQuery) ([]ForumMessage, error) {
	constraints := sq.And{sq.Eq{"forum_thread_id": filters.ForumThreadID}}

	builder := f.db.
		Builder().
		Select("m.forum_message_id", "m.forum_thread_id", "m.source_id", "m.body_md", "m.created_on",
			"m.updated_on", "p.personaname", "p.avatarhash", "p.permission_level", "coalesce(s.forum_signature, '')").
		From("forum_message m").
		LeftJoin("person p ON p.steam_id = m.source_id").
		LeftJoin("person_settings s ON s.steam_id = m.source_id").
		Where(constraints).
		OrderBy("m.forum_message_id")

	rows, errRows := f.db.QueryBuilder(ctx, nil, builder.Where(constraints))
	if errRows != nil {
		return nil, database.DBErr(errRows)
	}
	defer rows.Close()

	var messages []ForumMessage

	for rows.Next() {
		var msg ForumMessage
		if errScan := rows.Scan(&msg.ForumMessageID, &msg.ForumThreadID, &msg.SourceID, &msg.BodyMD, &msg.CreatedOn, &msg.UpdatedOn,
			&msg.Personaname, &msg.Avatarhash, &msg.PermissionLevel, &msg.Signature); errScan != nil {
			return nil, database.DBErr(errScan)
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func (f *ForumRepository) ForumMessageDelete(ctx context.Context, messageID int64) error {
	return database.DBErr(f.db.ExecDeleteBuilder(ctx, nil, f.db.
		Builder().
		Delete("forum_message").
		Where(sq.Eq{"forum_message_id": messageID})))
}

func (f *ForumRepository) ForumMessageVoteApply(ctx context.Context, messageVote *ForumMessageVote) error {
	var existingVote ForumMessageVote

	row, errRow := f.db.QueryRowBuilder(ctx, nil, f.db.
		Builder().
		Select("forum_message_vote_id", "forum_message_id", "source_id", "vote", "created_on", "updated_on").
		From("forum_message_vote").
		Where(sq.And{sq.Eq{"forum_message_id": messageVote.ForumMessageID}, sq.Eq{"source_id": messageVote.SourceID.Int64()}}))
	if errRow != nil {
		if !errors.Is(errRow, database.ErrNoResult) {
			return database.DBErr(errRow)
		}
	}

	errScan := database.DBErr(row.Scan(&existingVote.ForumMessageVoteID, &existingVote.ForumMessageID, &existingVote.SourceID,
		&existingVote.Vote, &existingVote.CreatedOn, &existingVote.UpdatedOn))
	if errScan != nil {
		if !errors.Is(errScan, database.ErrNoResult) {
			return database.DBErr(errScan)
		}
	}

	// If the vote exists and is the same vote, delete it. Otherwise, update the existing vote
	if existingVote.ForumMessageVoteID > 0 {
		if existingVote.Vote == messageVote.Vote {
			return database.DBErr(f.db.ExecDeleteBuilder(ctx, nil, f.db.
				Builder().
				Delete("forum_message_vote").
				Where(sq.Eq{"forum_message_vote_id": existingVote.ForumMessageVoteID})))
		}

		return database.DBErr(f.db.ExecUpdateBuilder(ctx, nil, f.db.
			Builder().
			Update("forum_message_vote").
			Set("vote", messageVote.Vote).
			Where(sq.Eq{"forum_message_vote_id": existingVote.ForumMessageVoteID})))
	}

	return database.DBErr(f.db.ExecInsertBuilderWithReturnValue(ctx, nil, f.db.
		Builder().
		Insert("forum_message_vote").
		SetMap(map[string]interface{}{
			"forum_message_id": messageVote.ForumMessageID,
			"source_id":        messageVote.SourceID.Int64(),
			"vote":             messageVote.Vote,
			"created_on":       messageVote.CreatedOn,
			"updated_on":       messageVote.UpdatedOn,
		}).
		Suffix("RETURNING forum_message_vote_id"), &messageVote.ForumMessageVoteID))
}

func (f *ForumRepository) ForumMessageVoteByID(ctx context.Context, messageVoteID int64, messageVote *ForumMessageVote) error {
	row, errRow := f.db.QueryRowBuilder(ctx, nil, f.db.
		Builder().
		Select("forum_message_vote_id", "forum_message_id", "source_id", "vote", "created_on", "updated_on").
		From("forum_message_vote").Where(sq.Eq{"forum_message_vote_id": messageVoteID}))
	if errRow != nil {
		return database.DBErr(errRow)
	}

	return database.DBErr(row.Scan(&messageVote.ForumMessageVoteID, &messageVote.ForumMessageID,
		&messageVote.SourceID, &messageVote.Vote, &messageVote.CreatedOn, &messageVote.UpdatedOn))
}
