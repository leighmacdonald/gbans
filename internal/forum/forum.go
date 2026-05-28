package forum

// - Categories
//   - Forums
//     - Threads
// 		 - Messages
import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain/person"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/leighmacdonald/gbans/pkg/stringutil"
	"github.com/leighmacdonald/steamid/v4/steamid"
)

var (
	ErrInvalidThread = errors.New("invalid thread id")
	ErrThreadLocked  = errors.New("thread is locked")
)

type ThreadMessagesQuery struct {
	Deleted       bool
	ForumThreadID int32
}

type ThreadQueryFilter struct {
	ForumID int32
}

type Activity struct {
	Person       person.BaseUser
	LastActivity time.Time
}

func (activity Activity) Expired() bool {
	return time.Since(activity.LastActivity) > time.Minute*5
}

type Category struct {
	ForumCategoryID int32
	Title           string
	Description     string
	Ordering        int32
	Forums          []Forum
	CreatedOn       time.Time
	UpdatedOn       time.Time
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
	ForumID             int32
	ForumCategoryID     int32
	LastThreadID        int32
	Title               string
	Description         string
	Ordering            int32
	CountThreads        int32
	CountMessages       int32
	PermissionLevel     permission.Privilege
	RecentForumThreadID int32
	RecentForumTitle    string
	RecentSourceID      string
	RecentPersonaname   string
	RecentAvatarhash    string
	RecentCreatedOn     time.Time
	CreatedOn           time.Time
	UpdatedOn           time.Time
}

func (f Forum) Path() string {
	return fmt.Sprintf("/forums/%d", f.ForumID)
}

func (f Forum) NewThread(title string, sourceID steamid.SteamID) Thread {
	return Thread{
		ForumID:   f.ForumID,
		SourceID:  sourceID,
		Title:     title,
		CreatedOn: time.Now(),
		UpdatedOn: time.Now(),
	}
}

type Thread struct {
	ForumThreadID   int32
	ForumID         int32
	SourceID        steamid.SteamID
	Title           string
	Sticky          bool
	Locked          bool
	Views           int32
	Replies         int32
	Personaname     string
	Avatarhash      string
	PermissionLevel permission.Privilege
	CreatedOn       time.Time
	UpdatedOn       time.Time
}

func (t Thread) Path() string {
	return fmt.Sprintf("/forums/thread/%d", t.ForumThreadID)
}

func (t Thread) NewMessage(sourceID steamid.SteamID, body string) Message {
	return Message{
		ForumMessageID: 0,
		ForumThreadID:  t.ForumThreadID,
		SourceID:       sourceID,
		BodyMD:         stringutil.SanitizeUGC(body),
		CreatedOn:      time.Now(),
		UpdatedOn:      time.Now(),
	}
}

type Message struct {
	ForumMessageID  int64
	ForumThreadID   int32
	SourceID        steamid.SteamID
	BodyMD          string
	Title           string
	Online          bool
	Signature       string
	Personaname     string
	Avatarhash      string
	PermissionLevel permission.Privilege
	CreatedOn       time.Time
	UpdatedOn       time.Time
}

func (m Message) Path() string {
	return fmt.Sprintf("/forums/thread/%d/#%d", m.ForumThreadID, m.ForumMessageID)
}

func (m Message) NewVote(sourceID steamid.SteamID, vote Vote) MessageVote {
	return MessageVote{
		ForumMessageID: m.ForumMessageID,
		SourceID:       sourceID,
		Vote:           vote,
		CreatedOn:      time.Now(),
		UpdatedOn:      time.Now(),
	}
}

type MessageVote struct {
	ForumMessageVoteID int64
	ForumMessageID     int64
	SourceID           steamid.SteamID
	Vote               Vote // -1/+1
	CreatedOn          time.Time
	UpdatedOn          time.Time
}

type ThreadWithSource struct {
	Thread

	Personaname          string
	Avatarhash           string
	PermissionLevel      permission.Privilege
	RecentForumMessageID int64
	RecentCreatedOn      time.Time
	RecentSteamID        int64
	RecentPersonaname    string
	RecentAvatarhash     string
}

type Vote int

const (
	VoteUp   = 1
	VoteNone = 0
	VoteDown = -1
)

type Forums struct {
	repo      Repository
	tracker   *Tracker
	persons   person.Provider
	notif     notification.Notifier
	channelID string
}

func New(repository Repository, notif notification.Notifier, persons person.Provider, channelID string) Forums {
	return Forums{repo: repository, tracker: NewTracker(), notif: notif, persons: persons, channelID: channelID}
}

func (f Forums) Start(ctx context.Context) {
	f.tracker.Start(ctx)
}

func (f Forums) Touch(up person.BaseUser) {
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

	go f.notif.Send(notification.NewDiscord(f.channelID, discordCategorySave(*category)))

	return nil
}

func (f Forums) Category(ctx context.Context, categoryID int32, category *Category) error {
	return f.repo.ForumCategory(ctx, categoryID, category)
}

func (f Forums) CategoryDelete(ctx context.Context, category Category) error {
	if err := f.repo.ForumCategoryDelete(ctx, category.ForumCategoryID); err != nil {
		return err
	}

	go f.notif.Send(notification.NewDiscord(f.channelID, discordCategoryDelete(category)))
	slog.Info("Forum category deleted", slog.String("category", category.Title), slog.Int("forum_category_id", int(category.ForumCategoryID)))

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

	f.notif.Send(notification.NewDiscord(f.channelID, discordForumSaved(*forum)))

	if isNew {
		slog.Info("New forum created", slog.String("title", forum.Title))
	} else {
		slog.Info("Forum updated", slog.String("title", forum.Title), slog.Int("forum_id", int(forum.ForumID)))
	}

	return nil
}

func (f Forums) Forum(ctx context.Context, forumID int32, forum *Forum) error {
	return f.repo.Forum(ctx, forumID, forum)
}

func (f Forums) ForumDelete(ctx context.Context, forumID int32) error {
	if err := f.repo.ForumDelete(ctx, forumID); err != nil {
		return err
	}

	slog.Info("Forum deleted successfully", slog.Int("forum_id", int(forumID)))

	return nil
}

func (f Forums) ThreadSave(ctx context.Context, thread *Thread) error {
	isNew := thread.ForumThreadID == 0

	if err := f.repo.ForumThreadSave(ctx, thread); err != nil {
		return err
	}

	if isNew {
		slog.Info("Thread created", slog.String("title", thread.Title), slog.Int("thread_id", int(thread.ForumThreadID)))
	} else {
		slog.Info("Forum thread updates", slog.String("title", thread.Title), slog.Int("thread_id", int(thread.ForumThreadID)))
	}

	return nil
}

func (f Forums) Thread(ctx context.Context, forumThreadID int32, thread *Thread) error {
	return f.repo.ForumThread(ctx, forumThreadID, thread)
}

func (f Forums) ThreadIncrView(ctx context.Context, forumThreadID int32) error {
	return f.repo.ForumThreadIncrView(ctx, forumThreadID)
}

func (f Forums) ThreadDelete(ctx context.Context, forumThreadID int32) error {
	if err := f.repo.ForumThreadDelete(ctx, forumThreadID); err != nil {
		return err
	}

	slog.Info("Forum thread deleted", slog.Int("forum_thread_id", int(forumThreadID)))

	return nil
}

func (f Forums) Threads(ctx context.Context, filter ThreadQueryFilter) ([]ThreadWithSource, error) {
	return f.repo.ForumThreads(ctx, filter)
}

func (f Forums) ForumIncrMessageCount(ctx context.Context, forumID int32, incr bool) error {
	return f.repo.ForumIncrMessageCount(ctx, forumID, incr)
}

type parents struct {
	Thread   Thread
	Forum    Forum
	Category Category
}

func (f Forums) getParents(ctx context.Context, forumThreadID int32) (parents, error) {
	var thread Thread
	if err := f.Thread(ctx, forumThreadID, &thread); err != nil {
		return parents{}, err
	}

	var forum Forum
	if err := f.Forum(ctx, thread.ForumID, &forum); err != nil {
		return parents{}, err
	}

	var category Category
	if err := f.Category(ctx, forum.ForumCategoryID, &category); err != nil {
		return parents{}, err
	}

	return parents{Thread: thread, Forum: forum, Category: category}, nil
}

func (f Forums) MessageSave(ctx context.Context, fMessage *Message) error {
	isNew := fMessage.ForumMessageID == 0

	if err := f.repo.ForumMessageSave(ctx, fMessage); err != nil {
		return err
	}

	parent, errParents := f.getParents(ctx, fMessage.ForumThreadID)
	if errParents != nil {
		return errParents
	}

	author, errAuthor := f.persons.GetOrCreatePersonBySteamID(ctx, fMessage.SourceID)
	if errAuthor != nil {
		return errAuthor
	}

	go f.notif.Send(notification.NewDiscord(f.channelID,
		discordForumMessageSaved(parent, author, fMessage)))

	if isNew {
		if errIncr := f.ForumIncrMessageCount(ctx, parent.Forum.ForumID, true); errIncr != nil {
			return errIncr
		}

		slog.Info("Created new forum message", slog.Int("forum_thread_id", int(fMessage.ForumThreadID)))
	} else {
		slog.Info("Forum message edited", slog.Int("forum_thread_id", int(fMessage.ForumThreadID)))
	}

	return nil
}

func (f Forums) RecentActivity(ctx context.Context, limit uint64, permissionLevel permission.Privilege) ([]Message, error) {
	return f.repo.ForumRecentActivity(ctx, limit, permissionLevel)
}

func (f Forums) Message(ctx context.Context, messageID int64, forumMessage *Message) error {
	if messageID <= 0 {
		return database.ErrNoResult
	}

	return f.repo.ForumMessage(ctx, messageID, forumMessage)
}

func (f Forums) Messages(ctx context.Context, filters ThreadMessagesQuery) ([]Message, error) {
	return f.repo.ForumMessages(ctx, filters)
}

func (f Forums) MessageDelete(ctx context.Context, person person.BaseUser, messageID int64) error {
	var message Message
	if err := f.Message(ctx, messageID, &message); err != nil {
		return err
	}

	var thread Thread
	if err := f.Thread(ctx, message.ForumThreadID, &thread); err != nil {
		return err
	}

	if thread.Locked {
		return ErrThreadLocked
	}

	if !httphelper.HasPrivilege(person, steamid.Collection{message.SourceID}, permission.Editor) {
		return permission.ErrDenied
	}

	messages, errMessage := f.Messages(ctx, ThreadMessagesQuery{ForumThreadID: message.ForumThreadID})
	if errMessage != nil {
		return errMessage
	}

	isThreadParent := messages[0].ForumMessageID == message.ForumMessageID

	if isThreadParent { //nolint:nestif
		if err := f.ThreadDelete(ctx, message.ForumThreadID); err != nil {
			return err
		}

		// Delete the thread if it's the first message
		var forum Forum
		if errForum := f.Forum(ctx, thread.ForumID, &forum); errForum != nil {
			return errForum
		}

		forum.CountThreads--

		if errSave := f.ForumSave(ctx, &forum); errSave != nil {
			return errSave
		}

		slog.Error("Thread deleted due to parent deletion", slog.Int("forum_thread_id", int(thread.ForumThreadID)))
	} else {
		if errDelete := f.MessageDelete(ctx, person, message.ForumMessageID); errDelete != nil {
			return errDelete
		}
	}

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

func (t *Tracker) Touch(person person.BaseUser) {
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

	activity := t.activity

	return activity
}
