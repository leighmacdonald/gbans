package domain

import (
	"context"
	"time"

	"github.com/leighmacdonald/gbans/pkg/util"
	"github.com/leighmacdonald/steamid/v3/steamid"
)

type ForumRepository interface {
	Current() []ForumActivity
	Touch(up UserProfile)
	ForumCategories(ctx context.Context) ([]ForumCategory, error)
	ForumCategorySave(ctx context.Context, category *ForumCategory) error
	ForumCategory(ctx context.Context, categoryID int, category *ForumCategory) error
	ForumCategoryDelete(ctx context.Context, categoryID int) error
	Forums(ctx context.Context) ([]Forum, error)
	ForumSave(ctx context.Context, forum *Forum) error
	Forum(ctx context.Context, forumID int, forum *Forum) error
	ForumDelete(ctx context.Context, forumID int) error
	ForumThreadSave(ctx context.Context, thread *ForumThread) error
	ForumThread(ctx context.Context, forumThreadID int64, thread *ForumThread) error
	ForumThreadIncrView(ctx context.Context, forumThreadID int64) error
	ForumThreadDelete(ctx context.Context, forumThreadID int64) error
	ForumThreads(ctx context.Context, filter ThreadQueryFilter) ([]ThreadWithSource, int64, error)
	ForumIncrMessageCount(ctx context.Context, forumID int, incr bool) error
	ForumMessageSave(ctx context.Context, message *ForumMessage) error
	ForumRecentActivity(ctx context.Context, limit uint64, permissionLevel Privilege) ([]ForumMessage, error)
	ForumMessage(ctx context.Context, messageID int64, forumMessage *ForumMessage) error
	ForumMessages(ctx context.Context, filters ThreadMessagesQueryFilter) ([]ForumMessage, int64, error)
	ForumMessageDelete(ctx context.Context, messageID int64) error
	ForumMessageVoteApply(ctx context.Context, messageVote *ForumMessageVote) error
	ForumMessageVoteByID(ctx context.Context, messageVoteID int64, messageVote *ForumMessageVote) error
}

type ForumUsecase interface {
	Current() []ForumActivity
	Touch(up UserProfile)
	ForumCategories(ctx context.Context) ([]ForumCategory, error)
	ForumCategorySave(ctx context.Context, category *ForumCategory) error
	ForumCategory(ctx context.Context, categoryID int, category *ForumCategory) error
	ForumCategoryDelete(ctx context.Context, categoryID int) error
	Forums(ctx context.Context) ([]Forum, error)
	ForumSave(ctx context.Context, forum *Forum) error
	Forum(ctx context.Context, forumID int, forum *Forum) error
	ForumDelete(ctx context.Context, forumID int) error
	ForumThreadSave(ctx context.Context, thread *ForumThread) error
	ForumThread(ctx context.Context, forumThreadID int64, thread *ForumThread) error
	ForumThreadIncrView(ctx context.Context, forumThreadID int64) error
	ForumThreadDelete(ctx context.Context, forumThreadID int64) error
	ForumThreads(ctx context.Context, filter ThreadQueryFilter) ([]ThreadWithSource, int64, error)
	ForumIncrMessageCount(ctx context.Context, forumID int, incr bool) error
	ForumMessageSave(ctx context.Context, message *ForumMessage) error
	ForumRecentActivity(ctx context.Context, limit uint64, permissionLevel Privilege) ([]ForumMessage, error)
	ForumMessage(ctx context.Context, messageID int64, forumMessage *ForumMessage) error
	ForumMessages(ctx context.Context, filters ThreadMessagesQueryFilter) ([]ForumMessage, int64, error)
	ForumMessageDelete(ctx context.Context, messageID int64) error
	ForumMessageVoteApply(ctx context.Context, messageVote *ForumMessageVote) error
	ForumMessageVoteByID(ctx context.Context, messageVoteID int64, messageVote *ForumMessageVote) error
}

type ForumActivity struct {
	Person       UserProfile
	LastActivity time.Time
}

func (activity ForumActivity) Expired() bool {
	return time.Since(activity.LastActivity) > time.Minute*5
}

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
	ForumID             int           `json:"forum_id"`
	ForumCategoryID     int           `json:"forum_category_id"`
	LastThreadID        int64         `json:"last_thread_id"`
	Title               string        `json:"title"`
	Description         string        `json:"description"`
	Ordering            int           `json:"ordering"`
	CountThreads        int64         `json:"count_threads"`
	CountMessages       int64         `json:"count_messages"`
	PermissionLevel     Privilege     `json:"permission_level"`
	RecentForumThreadID int64         `json:"recent_forum_thread_id"`
	RecentForumTitle    string        `json:"recent_forum_title"`
	RecentSourceID      steamid.SID64 `json:"recent_source_id"`
	RecentPersonaname   string        `json:"recent_personaname"`
	RecentAvatarhash    string        `json:"recent_avatarhash"`
	RecentCreatedOn     time.Time     `json:"recent_created_on"`
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
		BodyMD:         util.SanitizeUGC(body),
		TimeStamped:    NewTimeStamped(),
	}
}

type ForumMessage struct {
	ForumMessageID int64         `json:"forum_message_id"`
	ForumThreadID  int64         `json:"forum_thread_id"`
	SourceID       steamid.SID64 `json:"source_id"`
	BodyMD         string        `json:"body_md"`
	Title          string        `json:"title"`
	Online         bool          `json:"online"`
	Signature      string        `json:"signature"`
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

type ThreadWithSource struct {
	ForumThread
	SimplePerson
	RecentForumMessageID int64     `json:"recent_forum_message_id"`
	RecentCreatedOn      time.Time `json:"recent_created_on"`
	RecentSteamID        string    `json:"recent_steam_id"`
	RecentPersonaname    string    `json:"recent_personaname"`
	RecentAvatarhash     string    `json:"recent_avatarhash"`
}

type Vote int

const (
	VoteUp   = 1
	VoteNone = 0
	VoteDown = -1
)
