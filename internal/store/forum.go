package store

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/leighmacdonald/steamid/v3/steamid"
	"github.com/pkg/errors"
)

type ForumCategory struct {
	ForumCategoryID int     `json:"forum_category_id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	Ordering        int     `json:"ordering"`
	Forums          []Forum `json:"forums"`
	TimeStamped
}

func (category ForumCategory) NewForum(title string, description string) Forum {
	return Forum{
		ForumID:         0,
		ForumCategoryID: category.ForumCategoryID,
		LastThreadID:    0,
		Title:           title,
		Description:     description,
		Ordering:        0,
		TimeStamped:     NewTimeStamped(),
	}
}

type Forum struct {
	ForumID             int              `json:"forum_id"`
	ForumCategoryID     int              `json:"forum_category_id"`
	LastThreadID        int64            `json:"last_thread_id"`
	Title               string           `json:"title"`
	Description         string           `json:"description"`
	Ordering            int              `json:"ordering"`
	CountThreads        int64            `json:"count_threads"`
	CountMessages       int64            `json:"count_messages"`
	PermissionLevel     consts.Privilege `json:"permission_level"`
	RecentForumThreadId int64            `json:"recent_forum_thread_id"`
	RecentForumTitle    string           `json:"recent_forum_title"`
	RecentSourceID      steamid.SID64    `json:"recent_source_id"`
	RecentPersonaname   string           `json:"recent_personaname"`
	RecentAvatarHash    string           `json:"recent_avatarhash"`
	RecentCreatedOn     time.Time        `json:"recent_created_on"`
	TimeStamped
}

func (forum Forum) NewThread(title string, sourceID steamid.SID64) ForumThread {
	return ForumThread{
		ForumID:     forum.ForumID,
		SourceID:    sourceID,
		Title:       title,
		TimeStamped: NewTimeStamped(),
	}
}

type SimplePerson struct {
	Personaname     string           `json:"personaname"`
	AvatarHash      string           `json:"avatarhash"`
	PermissionLevel consts.Privilege `json:"permission_level"`
}

type ForumThread struct {
	ForumThreadID int64         `json:"forum_thread_id"`
	ForumID       int           `json:"forum_id"`
	SourceID      steamid.SID64 `json:"source_id"`
	Title         string        `json:"title"`
	Sticky        bool          `json:"sticky"`
	Locked        bool          `json:"locked"`
	Views         int64         `json:"views"`
	Replies       int64         `json:"replies"`
	SimplePerson
	TimeStamped
}

func (thread ForumThread) NewMessage(sourceID steamid.SID64, body string) ForumMessage {
	return ForumMessage{
		ForumMessageID: 0,
		ForumThreadID:  thread.ForumThreadID,
		SourceID:       sourceID,
		BodyMD:         body,
		TimeStamped:    NewTimeStamped(),
	}
}

type ForumMessage struct {
	ForumMessageID int64         `json:"forum_message_id"`
	ForumThreadID  int64         `json:"forum_thread_id"`
	SourceID       steamid.SID64 `json:"source_id"`
	BodyMD         string        `json:"body_md"`
	SimplePerson
	TimeStamped
}

func (message ForumMessage) NewVote(sourceID steamid.SID64, vote Vote) ForumMessageVote {
	return ForumMessageVote{
		ForumMessageID: message.ForumMessageID,
		SourceID:       sourceID,
		Vote:           vote,
		TimeStamped:    NewTimeStamped(),
	}
}

type ForumMessageVote struct {
	ForumMessageVoteID int64         `json:"forum_message_vote_id"`
	ForumMessageID     int64         `json:"forum_message_id"`
	SourceID           steamid.SID64 `json:"source_id"`
	Vote               Vote          `json:"vote"` // -1/+1
	TimeStamped
}

func (db *Store) ForumCategories(ctx context.Context) ([]ForumCategory, error) {
	rows, errRows := db.QueryBuilder(ctx,
		db.sb.Select("forum_category_id", "title", "description", "ordering", "updated_on", "created_on").
			From("forum_category").
			OrderBy("ordering"))
	if errRows != nil {
		return nil, errRows
	}

	defer rows.Close()

	var categories []ForumCategory

	for rows.Next() {
		var fc ForumCategory
		if errScan := rows.Scan(&fc.ForumCategoryID, &fc.Title, &fc.Description, &fc.Ordering, &fc.UpdatedOn, &fc.CreatedOn); errScan != nil {
			return nil, Err(errScan)
		}

		categories = append(categories, fc)
	}

	return categories, nil
}

func (db *Store) ForumCategorySave(ctx context.Context, category *ForumCategory) error {
	category.UpdatedOn = time.Now()
	if category.ForumCategoryID > 0 {
		return db.ExecUpdateBuilder(ctx, db.sb.
			Update("forum_category").
			SetMap(map[string]interface{}{
				"title":       category.Title,
				"description": category.Description,
				"ordering":    category.Ordering,
				"updated_on":  category.UpdatedOn,
			}).
			Where(sq.Eq{"forum_category_id": category.ForumCategoryID}))
	}

	category.CreatedOn = category.UpdatedOn

	return db.ExecInsertBuilderWithReturnValue(ctx, db.sb.
		Insert("forum_category").
		SetMap(map[string]interface{}{
			"title":       category.Title,
			"description": category.Description,
			"ordering":    category.Ordering,
			"created_on":  category.CreatedOn,
			"updated_on":  category.UpdatedOn,
		}).
		Suffix("RETURNING forum_category_id"), &category.ForumCategoryID)
}

func (db *Store) ForumCategory(ctx context.Context, categoryID int, category *ForumCategory) error {
	row, errRow := db.QueryRowBuilder(ctx, db.sb.
		Select("forum_category_id", "title", "description", "ordering", "created_on", "created_on").
		From("forum_category").
		Where(sq.Eq{"forum_category_id": categoryID}))
	if errRow != nil {
		return errRow
	}

	return Err(row.Scan(&category.ForumCategoryID, &category.Title, &category.Description, &category.Ordering,
		&category.CreatedOn, &category.UpdatedOn))
}

func (db *Store) ForumCategoryDelete(ctx context.Context, categoryID int) error {
	return db.ExecDeleteBuilder(ctx, db.sb.
		Delete("forum_category").
		Where(sq.Eq{"forum_category_id": categoryID}))
}

func (db *Store) Forums(ctx context.Context) ([]Forum, error) {
	rows, errRows := db.QueryBuilder(ctx, db.sb.
		Select("DISTINCT f.forum_id", "f.forum_category_id", "f.title", "f.description", "f.last_thread_id",
			"f.count_threads", "f.count_messages", "f.ordering", "f.created_on", "f.updated_on", "f.permission_level",
			"t.forum_thread_id", "t.source_id", "p.personaname", "p.avatarhash", "t.created_on", "t.title").
		From("forum f").
		LeftJoin("forum_thread t ON f.last_thread_id = t.forum_thread_id").
		LeftJoin("forum_message m ON t.forum_thread_id = m.forum_thread_id").
		LeftJoin("person p ON p.steam_id = m.source_id").
		OrderBy("f.ordering"))
	if errRows != nil {
		return nil, errRows
	}

	defer rows.Close()

	var forums []Forum

	for rows.Next() {
		var (
			lastID           *int64
			forum            Forum
			lastForumTheadId *int64
			lastSourceID     *steamid.SID64
			lastPersonaname  *string
			lastAvatarhash   *string
			lastCreatedOn    *time.Time
			lastTitle        *string
		)

		if errScan := rows.Scan(&forum.ForumID, &forum.ForumCategoryID, &forum.Title, &forum.Description,
			&lastID, &forum.CountThreads, &forum.CountMessages,
			&forum.Ordering, &forum.CreatedOn, &forum.UpdatedOn, &forum.PermissionLevel,
			&lastForumTheadId, &lastSourceID, &lastPersonaname, &lastAvatarhash,
			&lastCreatedOn, &lastTitle); errScan != nil {
			return nil, Err(errScan)
		}

		if lastID != nil {
			forum.LastThreadID = *lastID
		}

		if lastForumTheadId != nil {
			forum.RecentForumThreadId = *lastForumTheadId
			forum.RecentSourceID = *lastSourceID
			forum.RecentPersonaname = *lastPersonaname
			forum.RecentAvatarHash = *lastAvatarhash
			forum.RecentCreatedOn = *lastCreatedOn
			forum.RecentForumTitle = *lastTitle
		}

		forums = append(forums, forum)
	}

	return forums, nil
}

func (db *Store) ForumSave(ctx context.Context, forum *Forum) error {
	forum.UpdatedOn = time.Now()

	var lastThreadID *int64

	if forum.LastThreadID > 0 {
		lastThreadID = &forum.LastThreadID
	}

	if forum.ForumID > 0 {
		return db.ExecUpdateBuilder(ctx, db.sb.
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
			Where(sq.Eq{"forum_id": forum.ForumID}))
	}

	forum.CreatedOn = time.Now()

	return db.ExecInsertBuilderWithReturnValue(ctx, db.sb.
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
		Suffix("RETURNING forum_id"), &forum.ForumID)
}

func (db *Store) Forum(ctx context.Context, forumID int, forum *Forum) error {
	row, errRow := db.QueryRowBuilder(ctx, db.sb.
		Select("forum_id", "forum_category_id", "title", "description", "last_thread_id",
			"count_threads", "count_messages", "ordering", "created_on", "updated_on").
		From("forum").
		Where(sq.Eq{"forum_id": forumID}))
	if errRow != nil {
		return errRow
	}

	var lastThreadID *int64
	if err := row.Scan(&forum.ForumID, &forum.ForumCategoryID, &forum.Title, &forum.Description,
		&lastThreadID, &forum.CountThreads, &forum.CountMessages, &forum.Ordering,
		&forum.CreatedOn, &forum.UpdatedOn); err != nil {
		return Err(err)
	}

	if lastThreadID != nil {
		forum.LastThreadID = *lastThreadID
	}

	return nil
}

func (db *Store) ForumDelete(ctx context.Context, forumID int) error {
	return db.ExecDeleteBuilder(ctx, db.sb.
		Delete("forum").
		Where(sq.Eq{"forum_id": forumID}))
}

func (db *Store) ForumThreadSave(ctx context.Context, thread *ForumThread) error {
	thread.UpdatedOn = time.Now()
	if thread.ForumThreadID > 0 {
		return db.ExecUpdateBuilder(ctx, db.sb.
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
			Where(sq.Eq{"forum_thread_id": thread.ForumThreadID}))
	}

	thread.CreatedOn = time.Now()

	if errInsert := db.ExecInsertBuilderWithReturnValue(ctx, db.sb.
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
		return errInsert
	}

	return db.ExecUpdateBuilder(ctx, db.sb.Update("forum").
		Set("count_threads", sq.Expr("count_threads+1")).
		Set("last_thread_id", thread.ForumThreadID).
		Where(sq.Eq{"forum_id": thread.ForumID}))
}

func (db *Store) ForumThread(ctx context.Context, forumThreadID int64, thread *ForumThread) error {
	row, errRow := db.QueryRowBuilder(ctx, db.sb.
		Select("t.forum_thread_id", "t.forum_id", "t.source_id", "t.title", "t.sticky",
			"t.locked", "t.views", "t.created_on", "t.updated_on", "p.personaname", "p.avatarhash",
			"p.permission_level").
		From("forum_thread t").
		LeftJoin("person p ON p.steam_id = t.source_id").
		Where(sq.Eq{"t.forum_thread_id": forumThreadID}))
	if errRow != nil {
		return errRow
	}

	return Err(row.Scan(&thread.ForumThreadID, &thread.ForumID, &thread.SourceID, &thread.Title,
		&thread.Sticky, &thread.Locked, &thread.Views, &thread.CreatedOn,
		&thread.UpdatedOn, &thread.Personaname, &thread.AvatarHash, &thread.PermissionLevel))
}

func (db *Store) ForumThreadIncrView(ctx context.Context, forumThreadID int64) error {
	return db.ExecUpdateBuilder(ctx, db.sb.
		Update("forum_thread").
		Set("views", sq.Expr("views+1")).
		Where(sq.Eq{"forum_thread_id": forumThreadID}))
}

func (db *Store) ForumThreadDelete(ctx context.Context, forumThreadID int64) error {
	return db.ExecDeleteBuilder(ctx, db.sb.
		Delete("forum_thread").
		Where(sq.Eq{"forum_thread_id": forumThreadID}))
}

type ThreadWithSource struct {
	ForumThread
	SimplePerson
	RecentForumMessageID int64     `json:"recent_forum_message_id"`
	RecentCreatedOn      time.Time `json:"recent_created_on"`
	RecentSteamID        string    `json:"recent_steam_id"`
	RecentPersonaname    string    `json:"recent_personaname"`
	RecentAvatarHash     string    `json:"recent_avatarhash"`
}

type ThreadQueryFilter struct {
	QueryFilter
	ForumID int `json:"forum_id"`
}

func (db *Store) ForumThreads(ctx context.Context, filter ThreadQueryFilter) ([]ThreadWithSource, int64, error) {
	if filter.ForumID <= 0 {
		return nil, 0, errors.New("Invalid Thread")
	}

	constraints := sq.And{sq.Eq{"forum_id": filter.ForumID}}

	builder := db.sb.
		Select("t.forum_thread_id", "t.forum_id", "t.source_id", "t.title", "t.sticky",
			"t.locked", "t.views", "t.created_on", "t.updated_on", "p.personaname", "p.avatarhash", "p.permission_level",
			"a.steam_id", "a.personaname", "a.avatarhash", "m.forum_message_id", "m.created_on").
		From("forum_thread t").
		LeftJoin("person p ON p.steam_id = t.source_id").
		LeftJoin("forum_message m ON m.forum_message_id = t.last_forum_message_id").
		LeftJoin("person a ON m.source_id = a.steam_id")

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

	count, errCount := db.GetCount(ctx, db.sb.
		Select("COUNT(forum_thread_id)").
		From("forum_thread").
		Where(constraints))

	if errCount != nil {
		return nil, 0, errCount
	}

	if count == 0 {
		return []ThreadWithSource{}, 0, nil
	}

	rows, errRows := db.QueryBuilder(ctx, builder)
	if errRows != nil {
		return nil, 0, errRows
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
				&tws.Locked, &tws.Views, &tws.CreatedOn, &tws.UpdatedOn, &tws.Personaname, &tws.AvatarHash,
				&tws.PermissionLevel, &RecentSteamID, &RecentPersonaname, &RecentAvatarHash, &RecentForumMessageID,
				&RecentCreatedOn); errScan != nil {
			return nil, 0, Err(errScan)
		}

		if RecentForumMessageID != nil {
			tws.RecentForumMessageID = *RecentForumMessageID
			tws.RecentCreatedOn = *RecentCreatedOn
			tws.RecentSteamID = *RecentSteamID
			tws.Personaname = *RecentPersonaname
			tws.AvatarHash = *RecentAvatarHash
		}

		threads = append(threads, tws)
	}

	return threads, count, nil
}

func (db *Store) ForumIncrMessageCount(ctx context.Context, forumID int, incr bool) error {
	builder := db.sb.Update("forum").
		Where(sq.Eq{"forum_id": forumID})

	if incr {
		builder = builder.Set("count_messages", sq.Expr("count_messages+1"))
	} else {
		builder = builder.Set("count_messages", sq.Expr("count_messages-1"))
	}

	return db.ExecUpdateBuilder(ctx, builder)
}

func (db *Store) ForumMessageSave(ctx context.Context, message *ForumMessage) error {
	message.UpdatedOn = time.Now()
	if message.ForumMessageID > 0 {
		return db.ExecUpdateBuilder(ctx, db.sb.
			Update("forum_message").
			SetMap(map[string]interface{}{
				"forum_thread_id": message.ForumThreadID,
				"source_id":       message.SourceID.Int64(),
				"body_md":         message.BodyMD,
				"updated_on":      message.UpdatedOn,
			}).
			Where(sq.Eq{"forum_message_id": message.ForumMessageID}))
	}

	message.CreatedOn = time.Now()

	if errInsert := db.ExecInsertBuilderWithReturnValue(ctx, db.sb.
		Insert("forum_message").
		SetMap(map[string]interface{}{
			"forum_thread_id": message.ForumThreadID,
			"source_id":       message.SourceID.Int64(),
			"body_md":         message.BodyMD,
			"created_on":      message.CreatedOn,
			"updated_on":      message.UpdatedOn,
		}).
		Suffix("RETURNING forum_message_id"), &message.ForumMessageID); errInsert != nil {
		return errInsert
	}

	return db.ExecUpdateBuilder(ctx, db.sb.
		Update("forum_thread").
		Set("last_forum_message_id", message.ForumMessageID).
		Where(sq.Eq{"forum_thread_id": message.ForumThreadID}))

}

func (db *Store) ForumMessage(ctx context.Context, messageID int64, forumMessage *ForumMessage) error {
	row, errRow := db.QueryRowBuilder(ctx, db.sb.
		Select("m.forum_message_id", "m.forum_thread_id", "m.source_id", "m.body_md", "m.created_on", "m.updated_on",
			"p.personaname", "p.avatarhash", "p.permission_level").
		From("forum_message m").
		LeftJoin("person p ON p.steam_id = m.source_id").
		Where(sq.Eq{"forum_message_id": messageID}))
	if errRow != nil {
		return errRow
	}

	return Err(row.Scan(&forumMessage.ForumMessageID, &forumMessage.ForumThreadID, &forumMessage.SourceID,
		&forumMessage.BodyMD, &forumMessage.CreatedOn, &forumMessage.UpdatedOn, &forumMessage.Personaname,
		&forumMessage.AvatarHash, &forumMessage.PermissionLevel))
}

type ThreadMessagesQueryFilter struct {
	QueryFilter
	ForumThreadID int64 `json:"forum_thread_id"`
}

func (db *Store) ForumMessages(ctx context.Context, filters ThreadMessagesQueryFilter) ([]ForumMessage, int64, error) {
	constraints := sq.And{sq.Eq{"forum_thread_id": filters.ForumThreadID}}

	rows, errRows := db.QueryBuilder(ctx, db.sb.
		Select("m.forum_message_id", "m.forum_thread_id", "m.source_id", "m.body_md", "m.created_on",
			"m.updated_on", "p.personaname", "p.avatarhash", "p.permission_level").
		From("forum_message m").
		LeftJoin("person p ON p.steam_id = m.source_id").
		Where(constraints))
	if errRows != nil {
		return nil, 0, errRows
	}

	defer rows.Close()

	var messages []ForumMessage

	for rows.Next() {
		var m ForumMessage
		if errScan := rows.Scan(&m.ForumMessageID, &m.ForumThreadID, &m.SourceID, &m.BodyMD, &m.CreatedOn, &m.UpdatedOn,
			&m.Personaname, &m.AvatarHash, &m.PermissionLevel); errScan != nil {
			return nil, 0, Err(errScan)
		}

		messages = append(messages, m)
	}

	count, errCount := db.GetCount(ctx, db.sb.
		Select("COUNT(m.forum_message_id)").
		From("forum_message m").
		Where(constraints))

	if errCount != nil {
		return nil, 0, errCount
	}

	return messages, count, nil
}

func (db *Store) ForumMessageDelete(ctx context.Context, messageID int64) error {
	return db.ExecDeleteBuilder(ctx, db.sb.
		Delete("forum_message").
		Where(sq.Eq{"forum_message_id": messageID}))
}

type Vote int

const (
	VoteUp   = 1
	VoteNone = 0
	VoteDown = -1
)

func (db *Store) ForumMessageVoteApply(ctx context.Context, messageVote *ForumMessageVote) error {
	var existingVote ForumMessageVote

	row, errRow := db.QueryRowBuilder(ctx, db.sb.
		Select("forum_message_vote_id", "forum_message_id", "source_id", "vote", "created_on", "updated_on").
		From("forum_message_vote").
		Where(sq.And{sq.Eq{"forum_message_id": messageVote.ForumMessageID}, sq.Eq{"source_id": messageVote.SourceID.Int64()}}))
	if errRow != nil {
		if !errors.Is(errRow, ErrNoResult) {
			return errRow
		}
	}

	errScan := Err(row.Scan(&existingVote.ForumMessageVoteID, &existingVote.ForumMessageID, &existingVote.SourceID,
		&existingVote.Vote, &existingVote.CreatedOn, &existingVote.UpdatedOn))
	if errScan != nil {
		if !errors.Is(errScan, ErrNoResult) {
			return Err(errScan)
		}
	}

	// If the vote exists and is the same vote, delete it. Otherwise, update the existing vote
	if existingVote.ForumMessageVoteID > 0 {
		if existingVote.Vote == messageVote.Vote {
			return db.ExecDeleteBuilder(ctx, db.sb.
				Delete("forum_message_vote").
				Where(sq.Eq{"forum_message_vote_id": existingVote.ForumMessageVoteID}))
		} else {
			return db.ExecUpdateBuilder(ctx, db.sb.
				Update("forum_message_vote").
				Set("vote", messageVote.Vote).
				Where(sq.Eq{"forum_message_vote_id": existingVote.ForumMessageVoteID}))
		}
	}

	return db.ExecInsertBuilderWithReturnValue(ctx, db.sb.
		Insert("forum_message_vote").
		SetMap(map[string]interface{}{
			"forum_message_id": messageVote.ForumMessageID,
			"source_id":        messageVote.SourceID.Int64(),
			"vote":             messageVote.Vote,
			"created_on":       messageVote.CreatedOn,
			"updated_on":       messageVote.UpdatedOn,
		}).
		Suffix("RETURNING forum_message_vote_id"), &messageVote.ForumMessageVoteID)
}

func (db *Store) ForumMessageVoteByID(ctx context.Context, messageVoteID int64, messageVote *ForumMessageVote) error {
	row, errRow := db.QueryRowBuilder(ctx, db.sb.
		Select("forum_message_vote_id", "forum_message_id", "source_id", "vote", "created_on", "updated_on").
		From("forum_message_vote").Where(sq.Eq{"forum_message_vote_id": messageVoteID}))
	if errRow != nil {
		return errRow
	}

	return Err(row.Scan(&messageVote.ForumMessageVoteID, &messageVote.ForumMessageID,
		&messageVote.SourceID, &messageVote.Vote, &messageVote.CreatedOn, &messageVote.UpdatedOn))
}
