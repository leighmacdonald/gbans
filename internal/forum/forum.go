package forum

// - Categories
//   - Forums
//     - Threads
// 		 - Messages
import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrInvalidThread = errors.New("invalid thread id")
	ErrThreadLocked  = errors.New("thread is locked")
)

type ThreadMessagesQuery struct {
	Deleted       bool  `json:"deleted,omitempty" uri:"deleted"`
	ForumThreadID int64 `json:"forum_thread_id"`
}

type ThreadQueryFilter struct {
	ForumID int `json:"forum_id"`
}

type Activity struct {
	Person       person.Info
	LastActivity time.Time
}

func (activity Activity) Expired() bool {
	return time.Since(activity.LastActivity) > time.Minute*5
}

type Category struct {
	ForumCategoryID int       `json:"forum_category_id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Ordering        int       `json:"ordering"`
	Forums          []Forum   `json:"forums"`
	CreatedOn       time.Time `json:"created_on"`
	UpdatedOn       time.Time `json:"updated_on"`
}

func (category Category) NewForum(title string, description string) Forum {
	return Forum{
		ForumID:         0,
		ForumCategoryID: category.ForumCategoryID,
		LastThreadID:    0,
		Title:           title,
		Description:     description,
		Ordering:        0,
		CreatedOn:       time.Now(),
		UpdatedOn:       time.Now(),
	}
}

type Forum struct {
	ForumID             int                  `json:"forum_id"`
	ForumCategoryID     int                  `json:"forum_category_id"`
	LastThreadID        int64                `json:"last_thread_id"`
	Title               string               `json:"title"`
	Description         string               `json:"description"`
	Ordering            int                  `json:"ordering"`
	CountThreads        int64                `json:"count_threads"`
	CountMessages       int64                `json:"count_messages"`
	PermissionLevel     permission.Privilege `json:"permission_level"`
	RecentForumThreadID int64                `json:"recent_forum_thread_id"`
	RecentForumTitle    string               `json:"recent_forum_title"`
	RecentSourceID      string               `json:"recent_source_id"`
	RecentPersonaname   string               `json:"recent_personaname"`
	RecentAvatarhash    string               `json:"recent_avatarhash"`
	RecentCreatedOn     time.Time            `json:"recent_created_on"`
	CreatedOn           time.Time            `json:"created_on"`
	UpdatedOn           time.Time            `json:"updated_on"`
}

func (forum Forum) NewThread(title string, sourceID steamid.SteamID) Thread {
	return Thread{
		ForumID:   forum.ForumID,
		SourceID:  sourceID,
		Title:     title,
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
	}
}

type Thread struct {
	ForumThreadID   int64                `json:"forum_thread_id"`
	ForumID         int                  `json:"forum_id"`
	SourceID        steamid.SteamID      `json:"source_id"`
	Title           string               `json:"title"`
	Sticky          bool                 `json:"sticky"`
	Locked          bool                 `json:"locked"`
	Views           int64                `json:"views"`
	Replies         int64                `json:"replies"`
	Personaname     string               `json:"personaname"`
	Avatarhash      string               `json:"avatarhash"`
	PermissionLevel permission.Privilege `json:"permission_level"`
	CreatedOn       time.Time            `json:"created_on"`
	UpdatedOn       time.Time            `json:"updated_on"`
}

func (thread Thread) NewMessage(sourceID steamid.SteamID, body string) Message {
	return Message{
		ForumMessageID: 0,
		ForumThreadID:  thread.ForumThreadID,
		SourceID:       sourceID,
		BodyMD:         stringutil.SanitizeUGC(body),
		CreatedOn:      time.Now(),
		UpdatedOn:      time.Now(),
	}
}

type Message struct {
	ForumMessageID  int64                `json:"forum_message_id"`
	ForumThreadID   int64                `json:"forum_thread_id"`
	SourceID        steamid.SteamID      `json:"source_id"`
	BodyMD          string               `json:"body_md"`
	Title           string               `json:"title"`
	Online          bool                 `json:"online"`
	Signature       string               `json:"signature"`
	Personaname     string               `json:"personaname"`
	Avatarhash      string               `json:"avatarhash"`
	PermissionLevel permission.Privilege `json:"permission_level"`
	CreatedOn       time.Time            `json:"created_on"`
	UpdatedOn       time.Time            `json:"updated_on"`
}

func (message Message) NewVote(sourceID steamid.SteamID, vote Vote) MessageVote {
	return MessageVote{
		ForumMessageID: message.ForumMessageID,
		SourceID:       sourceID,
		Vote:           vote,
		CreatedOn:      time.Now(),
		UpdatedOn:      time.Now(),
	}
}

type MessageVote struct {
	ForumMessageVoteID int64           `json:"forum_message_vote_id"`
	ForumMessageID     int64           `json:"forum_message_id"`
	SourceID           steamid.SteamID `json:"source_id"`
	Vote               Vote            `json:"vote"` // -1/+1
	CreatedOn          time.Time       `json:"created_on"`
	UpdatedOn          time.Time       `json:"updated_on"`
}

type ThreadWithSource struct {
	Thread

	Personaname          string               `json:"personaname"`
	Avatarhash           string               `json:"avatarhash"`
	PermissionLevel      permission.Privilege `json:"permission_level"`
	RecentForumMessageID int64                `json:"recent_forum_message_id"`
	RecentCreatedOn      time.Time            `json:"recent_created_on"`
	RecentSteamID        string               `json:"recent_steam_id"`
	RecentPersonaname    string               `json:"recent_personaname"`
	RecentAvatarhash     string               `json:"recent_avatarhash"`
}

type Vote int

const (
	VoteUp   = 1
	VoteNone = 0
	VoteDown = -1
)

type Forums struct {
	repo    Repository
	tracker *Tracker
	notif   notification.Notifier
	config  *config.Configuration
}

func NewForums(repository Repository, config *config.Configuration, notif notification.Notifier) Forums {
	return Forums{repo: repository, tracker: NewTracker(), config: config, notif: notif}
}

func (f Forums) Start(ctx context.Context) {
	f.tracker.Start(ctx)
}

func (f Forums) Touch(up person.Info) {
	f.tracker.Touch(up)
}

func (f Forums) Current() []Activity {
	return f.tracker.Current()
}

func (f Forums) Categories(ctx context.Context) ([]Category, error) {
	return f.repo.ForumCategories(ctx)
}

func (f Forums) CategorySave(ctx context.Context, category *Category) error {
	isNew := category.ForumCategoryID == 0

	if err := f.repo.ForumCategorySave(ctx, category); err != nil {
		return err
	}

	if isNew {
		slog.Info("New forum category created", slog.String("title", category.Title))
	} else {
		slog.Info("Forum category updated", slog.String("title", category.Title))
	}

	f.notif.Send(notification.NewDiscord(f.config.Config().Discord.ForumLogChannelID, discordCategorySave(*category)))

	return nil
}

func (f Forums) Category(ctx context.Context, categoryID int, category *Category) error {
	return f.repo.ForumCategory(ctx, categoryID, category)
}

func (f Forums) CategoryDelete(ctx context.Context, category Category) error {
	if err := f.repo.ForumCategoryDelete(ctx, category.ForumCategoryID); err != nil {
		return err
	}

	f.notif.Send(notification.NewDiscord(f.config.Config().Discord.ForumLogChannelID, discordCategoryDelete(category)))
	slog.Info("Forum category deleted", slog.String("category", category.Title), slog.Int("forum_category_id", category.ForumCategoryID))

	return nil
}

func (f Forums) Forums(ctx context.Context) ([]Forum, error) {
	return f.repo.Forums(ctx)
}

func (f Forums) ForumSave(ctx context.Context, forum *Forum) error {
	isNew := forum.ForumID == 0

	if err := f.repo.ForumSave(ctx, forum); err != nil {
		return err
	}

	f.notif.Send(notification.NewDiscord(f.config.Config().Discord.ForumLogChannelID, discordForumSaved(*forum)))

	if isNew {
		slog.Info("New forum created", slog.String("title", forum.Title))
	} else {
		slog.Info("Forum updated", slog.String("title", forum.Title), slog.Int("forum_id", forum.ForumID))
	}

	return nil
}

func (f Forums) Forum(ctx context.Context, forumID int, forum *Forum) error {
	return f.repo.Forum(ctx, forumID, forum)
}

func (f Forums) ForumDelete(ctx context.Context, forumID int) error {
	if err := f.repo.ForumDelete(ctx, forumID); err != nil {
		return err
	}

	slog.Info("Forum deleted successfully", slog.Int("forum_id", forumID))

	return nil
}

func (f Forums) ThreadSave(ctx context.Context, thread *Thread) error {
	isNew := thread.ForumThreadID == 0

	if err := f.repo.ForumThreadSave(ctx, thread); err != nil {
		return err
	}

	if isNew {
		slog.Info("Thread created", slog.String("title", thread.Title), slog.Int64("thread_id", thread.ForumThreadID))
	} else {
		slog.Info("Forum thread updates", slog.String("title", thread.Title), slog.Int64("thread_id", thread.ForumThreadID))
	}

	return nil
}

func (f Forums) Thread(ctx context.Context, forumThreadID int64, thread *Thread) error {
	return f.repo.ForumThread(ctx, forumThreadID, thread)
}

func (f Forums) ThreadIncrView(ctx context.Context, forumThreadID int64) error {
	return f.repo.ForumThreadIncrView(ctx, forumThreadID)
}

func (f Forums) ThreadDelete(ctx context.Context, forumThreadID int64) error {
	if err := f.repo.ForumThreadDelete(ctx, forumThreadID); err != nil {
		return err
	}

	slog.Info("Forum thread deleted", slog.Int64("forum_thread_id", forumThreadID))

	return nil
}

func (f Forums) Threads(ctx context.Context, filter ThreadQueryFilter) ([]ThreadWithSource, error) {
	return f.repo.ForumThreads(ctx, filter)
}

func (f Forums) ForumIncrMessageCount(ctx context.Context, forumID int, incr bool) error {
	return f.repo.ForumIncrMessageCount(ctx, forumID, incr)
}

func (f Forums) MessageSave(ctx context.Context, fMessage *Message) error {
	isNew := fMessage.ForumMessageID == 0

	if err := f.repo.ForumMessageSave(ctx, fMessage); err != nil {
		return err
	}

	f.notif.Send(notification.NewDiscord(f.config.Config().Discord.ForumLogChannelID, discordForumMessageSaved(*fMessage)))

	if isNew {
		slog.Info("Created new forum message", slog.Int64("forum_thread_id", fMessage.ForumThreadID))
	} else {
		slog.Info("Forum message edited", slog.Int64("forum_thread_id", fMessage.ForumThreadID))
	}

	return nil
}

func (f Forums) RecentActivity(ctx context.Context, limit uint64, permissionLevel permission.Privilege) ([]Message, error) {
	return f.repo.ForumRecentActivity(ctx, limit, permissionLevel)
}

func (f Forums) Message(ctx context.Context, messageID int64, forumMessage *Message) error {
	return f.repo.ForumMessage(ctx, messageID, forumMessage)
}

func (f Forums) Messages(ctx context.Context, filters ThreadMessagesQuery) ([]Message, error) {
	return f.repo.ForumMessages(ctx, filters)
}

func (f Forums) MessageDelete(ctx context.Context, messageID int64) error {
	if err := f.repo.ForumMessageDelete(ctx, messageID); err != nil {
		return err
	}

	slog.Info("Forum message deleted", slog.Int64("message_id", messageID))

	return nil
}

func (f Forums) MessageVoteApply(ctx context.Context, messageVote *MessageVote) error {
	return f.repo.ForumMessageVoteApply(ctx, messageVote)
}

func (f Forums) MessageVoteByID(ctx context.Context, messageVoteID int64, messageVote *MessageVote) error {
	return f.repo.ForumMessageVoteByID(ctx, messageVoteID, messageVote)
}

type Tracker struct {
	activityMu sync.RWMutex
	activity   []Activity
}

func NewTracker() *Tracker {
	return &Tracker{
		activity: make([]Activity, 0),
	}
}

func (t *Tracker) Touch(person person.Info) {
	sid := person.GetSteamID()
	if !sid.Valid() {
		return
	}

	valid := []Activity{{LastActivity: time.Now(), Person: person}}

	t.activityMu.Lock()
	defer t.activityMu.Unlock()

	for _, activity := range t.activity {
		if activity.Person.GetSteamID() == sid {
			continue
		}

		valid = append(valid, activity)
	}

	t.activity = valid
}

func (t *Tracker) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 30)

	for {
		select {
		case <-ticker.C:
			var current []Activity

			t.activityMu.Lock()

			for _, entry := range t.activity {
				if entry.Expired() {
					continue
				}

				current = append(current, entry)
			}

			t.activity = current

			t.activityMu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

func (t *Tracker) Current() []Activity {
	t.activityMu.RLock()
	defer t.activityMu.RUnlock()

	var activity []Activity

	activity = append(activity, t.activity...)

	return activity
}
