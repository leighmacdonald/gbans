package store

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
)

func ForumCategories(ctx context.Context, db Store) ([]model.ForumCategory, error) {
	rows, errRows := db.QueryBuilder(ctx, db.
		Builder().
		Select("forum_category_id", "title", "description", "ordering", "updated_on", "created_on").
		From("forum_category").
		OrderBy("ordering"))
	if errRows != nil {
		return nil, DBErr(errRows)
	}

	defer rows.Close()

	var categories []model.ForumCategory

	for rows.Next() {
		var fc model.ForumCategory
		if errScan := rows.Scan(&fc.ForumCategoryID, &fc.Title, &fc.Description, &fc.Ordering, &fc.UpdatedOn, &fc.CreatedOn); errScan != nil {
			return nil, DBErr(errScan)
		}

		categories = append(categories, fc)
	}

	return categories, nil
}

func ForumCategorySave(ctx context.Context, database Store, category *model.ForumCategory) error {
	category.UpdatedOn = time.Now()
	if category.ForumCategoryID > 0 {
		return DBErr(database.ExecUpdateBuilder(ctx, database.
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

	return DBErr(database.ExecInsertBuilderWithReturnValue(ctx, database.
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

func ForumCategory(ctx context.Context, db Store, categoryID int, category *model.ForumCategory) error {
	row, errRow := db.QueryRowBuilder(ctx, db.
		Builder().
		Select("forum_category_id", "title", "description", "ordering", "created_on", "created_on").
		From("forum_category").
		Where(sq.Eq{"forum_category_id": categoryID}))
	if errRow != nil {
		return DBErr(errRow)
	}

	return DBErr(row.Scan(&category.ForumCategoryID, &category.Title, &category.Description, &category.Ordering,
		&category.CreatedOn, &category.UpdatedOn))
}

func ForumCategoryDelete(ctx context.Context, db Store, categoryID int) error {
	return DBErr(db.ExecDeleteBuilder(ctx, db.
		Builder().
		Delete("forum_category").
		Where(sq.Eq{"forum_category_id": categoryID})))
}

func Forums(ctx context.Context, database Store) ([]model.Forum, error) {
	fromSelect := database.
		Builder().
		Select("DISTINCT ON (f.forum_id) f.forum_id", "f.forum_category_id", "f.title", "f.description", "f.last_thread_id",
			"f.count_threads", "f.count_messages", "f.ordering", "f.created_on", "f.updated_on", "f.permission_level",
			"t.forum_thread_id", "t.source_id", "p.personaname", "p.avatarhash", "t.created_on", "t.title").
		From("forum f").
		LeftJoin("forum_thread t ON f.last_thread_id = t.forum_thread_id").
		LeftJoin("forum_message m ON t.forum_thread_id = m.forum_thread_id").
		LeftJoin("person p ON p.steam_id = m.source_id")

	rows, errRows := database.QueryBuilder(ctx, database.
		Builder().
		Select("x.*").
		FromSelect(fromSelect, "x").
		OrderBy("x.ordering"))
	if errRows != nil {
		return nil, DBErr(errRows)
	}

	defer rows.Close()

	var forums []model.Forum

	for rows.Next() {
		var (
			lastID           *int64
			forum            model.Forum
			lastForumTheadID *int64
			lastSourceID     *steamid.SID64
			lastPersonaname  *string
			lastAvatarhash   *string
			lastCreatedOn    *time.Time
			lastTitle        *string
		)

		if errScan := rows.Scan(&forum.ForumID, &forum.ForumCategoryID, &forum.Title, &forum.Description,
			&lastID, &forum.CountThreads, &forum.CountMessages,
			&forum.Ordering, &forum.CreatedOn, &forum.UpdatedOn, &forum.PermissionLevel,
			&lastForumTheadID, &lastSourceID, &lastPersonaname, &lastAvatarhash,
			&lastCreatedOn, &lastTitle); errScan != nil {
			return nil, DBErr(errScan)
		}

		if lastID != nil {
			forum.LastThreadID = *lastID
		}

		if lastForumTheadID != nil {
			forum.RecentForumThreadID = *lastForumTheadID
			forum.RecentSourceID = *lastSourceID
			forum.RecentPersonaname = *lastPersonaname
			forum.RecentAvatarhash = *lastAvatarhash
			forum.RecentCreatedOn = *lastCreatedOn
			forum.RecentForumTitle = *lastTitle
		}

		forums = append(forums, forum)
	}

	return forums, nil
}

func ForumSave(ctx context.Context, database Store, forum *model.Forum) error {
	forum.UpdatedOn = time.Now()

	var lastThreadID *int64

	if forum.LastThreadID > 0 {
		lastThreadID = &forum.LastThreadID
	}

	if forum.ForumID > 0 {
		return DBErr(database.ExecUpdateBuilder(ctx, database.
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

	return DBErr(database.ExecInsertBuilderWithReturnValue(ctx, database.
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

func Forum(ctx context.Context, db Store, forumID int, forum *model.Forum) error {
	row, errRow := db.QueryRowBuilder(ctx, db.
		Builder().
		Select("forum_id", "forum_category_id", "title", "description", "last_thread_id",
			"count_threads", "count_messages", "ordering", "created_on", "updated_on", "permission_level").
		From("forum").
		Where(sq.Eq{"forum_id": forumID}))
	if errRow != nil {
		return DBErr(errRow)
	}

	var lastThreadID *int64
	if err := row.Scan(&forum.ForumID, &forum.ForumCategoryID, &forum.Title, &forum.Description,
		&lastThreadID, &forum.CountThreads, &forum.CountMessages, &forum.Ordering,
		&forum.CreatedOn, &forum.UpdatedOn, &forum.PermissionLevel); err != nil {
		return DBErr(err)
	}

	if lastThreadID != nil {
		forum.LastThreadID = *lastThreadID
	}

	return nil
}

func ForumDelete(ctx context.Context, db Store, forumID int) error {
	return DBErr(db.ExecDeleteBuilder(ctx, db.
		Builder().
		Delete("forum").
		Where(sq.Eq{"forum_id": forumID})))
}

func ForumThreadSave(ctx context.Context, database Store, thread *model.ForumThread) error {
	thread.UpdatedOn = time.Now()
	if thread.ForumThreadID > 0 {
		return DBErr(database.ExecUpdateBuilder(ctx, database.
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

	if errInsert := database.ExecInsertBuilderWithReturnValue(ctx, database.
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
		return DBErr(errInsert)
	}

	return DBErr(database.ExecUpdateBuilder(ctx, database.
		Builder().
		Update("forum").
		Set("count_threads", sq.Expr("count_threads+1")).
		Set("last_thread_id", thread.ForumThreadID).
		Where(sq.Eq{"forum_id": thread.ForumID})))
}

func ForumThread(ctx context.Context, database Store, forumThreadID int64, thread *model.ForumThread) error {
	row, errRow := database.QueryRowBuilder(ctx, database.
		Builder().
		Select("t.forum_thread_id", "t.forum_id", "t.source_id", "t.title", "t.sticky",
			"t.locked", "t.views", "t.created_on", "t.updated_on", "p.personaname", "p.avatarhash",
			"p.permission_level").
		From("forum_thread t").
		LeftJoin("person p ON p.steam_id = t.source_id").
		Where(sq.Eq{"t.forum_thread_id": forumThreadID}))
	if errRow != nil {
		return DBErr(errRow)
	}

	return DBErr(row.Scan(&thread.ForumThreadID, &thread.ForumID, &thread.SourceID, &thread.Title,
		&thread.Sticky, &thread.Locked, &thread.Views, &thread.CreatedOn,
		&thread.UpdatedOn, &thread.Personaname, &thread.Avatarhash, &thread.PermissionLevel))
}

func ForumThreadIncrView(ctx context.Context, database Store, forumThreadID int64) error {
	return DBErr(database.ExecUpdateBuilder(ctx, database.
		Builder().
		Update("forum_thread").
		Set("views", sq.Expr("views+1")).
		Where(sq.Eq{"forum_thread_id": forumThreadID})))
}

func ForumThreadDelete(ctx context.Context, database Store, forumThreadID int64) error {
	return DBErr(database.ExecDeleteBuilder(ctx, database.
		Builder().
		Delete("forum_thread").
		Where(sq.Eq{"forum_thread_id": forumThreadID})))
}

type ThreadQueryFilter struct {
	QueryFilter
	ForumID int `json:"forum_id"`
}

func ForumThreads(ctx context.Context, database Store, filter ThreadQueryFilter) ([]model.ThreadWithSource, int64, error) {
	if filter.ForumID <= 0 {
		return nil, 0, errors.New("Invalid Thread")
	}

	constraints := sq.And{sq.Eq{"forum_id": filter.ForumID}}

	builder := database.
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
		OrderBy("t.sticky DESC, a.updated_on DESC")

	builder = filter.applySafeOrder(builder, map[string][]string{
		"t.": {
			"forum_thread_id", "forum_id", "source_id", "title", "sticky",
			"locked", "views", "created_on", "updated_on",
		},
		"p.": {"personaname", "avatar_hash", "permission_level"},
		"m.": {"created_on", "forum_message_id"},
		"a.": {"steam_id", "personaname", "avatarhash", "permission_level"},
	}, "short_name")

	builder = filter.applyLimitOffset(builder, 100).Where(constraints)

	count, errCount := getCount(ctx, database, database.
		Builder().
		Select("COUNT(forum_thread_id)").
		From("forum_thread").
		Where(constraints))

	if errCount != nil {
		return nil, 0, DBErr(errCount)
	}

	if count == 0 {
		return []model.ThreadWithSource{}, 0, nil
	}

	rows, errRows := database.QueryBuilder(ctx, builder)
	if errRows != nil {
		return nil, 0, DBErr(errRows)
	}

	defer rows.Close()

	var threads []model.ThreadWithSource

	for rows.Next() {
		var (
			tws                  model.ThreadWithSource
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
			return nil, 0, DBErr(errScan)
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

	return threads, count, nil
}

func ForumIncrMessageCount(ctx context.Context, database Store, forumID int, incr bool) error {
	builder := database.
		Builder().
		Update("forum").
		Where(sq.Eq{"forum_id": forumID})

	if incr {
		builder = builder.Set("count_messages", sq.Expr("count_messages+1"))
	} else {
		builder = builder.Set("count_messages", sq.Expr("count_messages-1"))
	}

	return DBErr(database.ExecUpdateBuilder(ctx, builder))
}

func ForumMessageSave(ctx context.Context, database Store, message *model.ForumMessage) error {
	message.UpdatedOn = time.Now()
	if message.ForumMessageID > 0 {
		return DBErr(database.ExecUpdateBuilder(ctx, database.
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

	if errInsert := database.ExecInsertBuilderWithReturnValue(ctx, database.
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
		return DBErr(errInsert)
	}

	return DBErr(database.ExecUpdateBuilder(ctx, database.
		Builder().
		Update("forum_thread").
		Set("updated_on", message.CreatedOn).
		Where(sq.Eq{"forum_thread_id": message.ForumThreadID})))
}

func ForumRecentActivity(ctx context.Context, database Store, limit uint64, permissionLevel consts.Privilege) ([]model.ForumMessage, error) {
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
				WHERE m.forum_thread_id = t.forum_thread_id AND $1 >= f.permission_level
				ORDER BY m.forum_message_id DESC
				LIMIT 1
				) m on TRUE`, permissionLevel).ToSql()
	if errExpr != nil {
		return nil, DBErr(errExpr)
	}

	builder := database.
		Builder().
		Select("t.forum_thread_id", "m.forum_message_id",
			"m.steam_id", "m.created_on", "m.updated_on", "m.personaname",
			"m.avatarhash", "f.permission_level", "t.title").
		From("forum_thread t").
		LeftJoin("forum f ON f.forum_id = t.forum_id").
		InnerJoin(expr).
		Where(sq.GtOrEq{"m.permission_level": permissionLevel}).
		OrderBy("t.updated_on DESC").
		Limit(limit)

	rows, errRows := database.QueryBuilder(ctx, builder)
	if errRows != nil {
		return nil, DBErr(errRows)
	}

	defer rows.Close()

	var messages []model.ForumMessage

	for rows.Next() {
		var msg model.ForumMessage
		if errScan := rows.Scan(&msg.ForumThreadID, &msg.ForumMessageID, &msg.SourceID,
			&msg.CreatedOn, &msg.UpdatedOn, &msg.Personaname,
			&msg.Avatarhash, &msg.PermissionLevel, &msg.Title); errScan != nil {
			return nil, DBErr(errScan)
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

func ForumMessage(ctx context.Context, db Store, messageID int64, forumMessage *model.ForumMessage) error {
	row, errRow := db.QueryRowBuilder(ctx, db.
		Builder().
		Select("m.forum_message_id", "m.forum_thread_id", "m.source_id", "m.body_md", "m.created_on", "m.updated_on",
			"p.personaname", "p.avatarhash", "p.permission_level", "coalesce(s.forum_signature, '')").
		From("forum_message m").
		LeftJoin("person p ON p.steam_id = m.source_id").
		LeftJoin("person_settings s ON s.steam_id = m.source_id").
		Where(sq.Eq{"forum_message_id": messageID}))
	if errRow != nil {
		return DBErr(errRow)
	}

	return DBErr(row.Scan(&forumMessage.ForumMessageID, &forumMessage.ForumThreadID, &forumMessage.SourceID,
		&forumMessage.BodyMD, &forumMessage.CreatedOn, &forumMessage.UpdatedOn, &forumMessage.Personaname,
		&forumMessage.Avatarhash, &forumMessage.PermissionLevel, &forumMessage.Signature))
}

type ThreadMessagesQueryFilter struct {
	QueryFilter
	ForumThreadID int64 `json:"forum_thread_id"`
}

func ForumMessages(ctx context.Context, database Store, filters ThreadMessagesQueryFilter) ([]model.ForumMessage, int64, error) {
	constraints := sq.And{sq.Eq{"forum_thread_id": filters.ForumThreadID}}

	builder := database.
		Builder().
		Select("m.forum_message_id", "m.forum_thread_id", "m.source_id", "m.body_md", "m.created_on",
			"m.updated_on", "p.personaname", "p.avatarhash", "p.permission_level", "coalesce(s.forum_signature, '')").
		From("forum_message m").
		LeftJoin("person p ON p.steam_id = m.source_id").
		LeftJoin("person_settings s ON s.steam_id = m.source_id").
		Where(constraints).
		OrderBy("m.forum_message_id")

	rows, errRows := database.QueryBuilder(ctx, filters.applyLimitOffset(builder, 100).Where(constraints))
	if errRows != nil {
		return nil, 0, DBErr(errRows)
	}
	defer rows.Close()

	var messages []model.ForumMessage

	for rows.Next() {
		var msg model.ForumMessage
		if errScan := rows.Scan(&msg.ForumMessageID, &msg.ForumThreadID, &msg.SourceID, &msg.BodyMD, &msg.CreatedOn, &msg.UpdatedOn,
			&msg.Personaname, &msg.Avatarhash, &msg.PermissionLevel, &msg.Signature); errScan != nil {
			return nil, 0, DBErr(errScan)
		}

		messages = append(messages, msg)
	}

	count, errCount := getCount(ctx, database, database.
		Builder().
		Select("COUNT(m.forum_message_id)").
		From("forum_message m").
		Where(constraints))

	if errCount != nil {
		return nil, 0, DBErr(errCount)
	}

	return messages, count, nil
}

func ForumMessageDelete(ctx context.Context, database Store, messageID int64) error {
	return DBErr(database.ExecDeleteBuilder(ctx, database.
		Builder().
		Delete("forum_message").
		Where(sq.Eq{"forum_message_id": messageID})))
}

func ForumMessageVoteApply(ctx context.Context, database Store, messageVote *model.ForumMessageVote) error {
	var existingVote model.ForumMessageVote

	row, errRow := database.QueryRowBuilder(ctx, database.
		Builder().
		Select("forum_message_vote_id", "forum_message_id", "source_id", "vote", "created_on", "updated_on").
		From("forum_message_vote").
		Where(sq.And{sq.Eq{"forum_message_id": messageVote.ForumMessageID}, sq.Eq{"source_id": messageVote.SourceID.Int64()}}))
	if errRow != nil {
		if !errors.Is(errRow, ErrNoResult) {
			return DBErr(errRow)
		}
	}

	errScan := DBErr(row.Scan(&existingVote.ForumMessageVoteID, &existingVote.ForumMessageID, &existingVote.SourceID,
		&existingVote.Vote, &existingVote.CreatedOn, &existingVote.UpdatedOn))
	if errScan != nil {
		if !errors.Is(errScan, ErrNoResult) {
			return DBErr(errScan)
		}
	}

	// If the vote exists and is the same vote, delete it. Otherwise, update the existing vote
	if existingVote.ForumMessageVoteID > 0 {
		if existingVote.Vote == messageVote.Vote {
			return DBErr(database.ExecDeleteBuilder(ctx, database.
				Builder().
				Delete("forum_message_vote").
				Where(sq.Eq{"forum_message_vote_id": existingVote.ForumMessageVoteID})))
		} else {
			return DBErr(database.ExecUpdateBuilder(ctx, database.
				Builder().
				Update("forum_message_vote").
				Set("vote", messageVote.Vote).
				Where(sq.Eq{"forum_message_vote_id": existingVote.ForumMessageVoteID})))
		}
	}

	return DBErr(database.ExecInsertBuilderWithReturnValue(ctx, database.
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

func ForumMessageVoteByID(ctx context.Context, database Store, messageVoteID int64, messageVote *model.ForumMessageVote) error {
	row, errRow := database.QueryRowBuilder(ctx, database.
		Builder().
		Select("forum_message_vote_id", "forum_message_id", "source_id", "vote", "created_on", "updated_on").
		From("forum_message_vote").Where(sq.Eq{"forum_message_vote_id": messageVoteID}))
	if errRow != nil {
		return DBErr(errRow)
	}

	return DBErr(row.Scan(&messageVote.ForumMessageVoteID, &messageVote.ForumMessageID,
		&messageVote.SourceID, &messageVote.Vote, &messageVote.CreatedOn, &messageVote.UpdatedOn))
}
