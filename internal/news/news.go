package news

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/httphelper"
)

type Article struct {
	NewsID           int32
	Title            string
	BodyMD           string
	IsPublished      bool
	CreatedOn        time.Time
	UpdatedOn        time.Time
	DiscordMessageID int64
}

type News struct {
	repository   Repository
	discord      discord.Connection
	logChannelID string
}

func New(repository Repository, discord discord.Connection, logChannelID string) News {
	return News{repository: repository, discord: discord, logChannelID: logChannelID}
}

func (u News) GetNewsLatest(ctx context.Context, limit int32, includeUnpublished bool) ([]Article, error) {
	return u.repository.GetNewsLatest(ctx, limit, includeUnpublished)
}

func (u News) GetNewsLatestArticle(ctx context.Context, includeUnpublished bool, entry *Article) error {
	return u.repository.GetNewsLatestArticle(ctx, includeUnpublished, entry)
}

func (u News) GetNewsByID(ctx context.Context, newsID int, entry *Article) error {
	return u.repository.GetNewsByID(ctx, newsID, entry)
}

func (u News) Save(ctx context.Context, entry *Article) error {
	if entry.Title == "" {
		return httphelper.ErrTooShort
	}

	if entry.BodyMD == "" {
		return httphelper.ErrTooShort
	}

	isNew := entry.NewsID == 0
	if isNew {
		now := time.Now()
		entry.CreatedOn = now
		entry.UpdatedOn = now
	}

	if err := u.repository.Save(ctx, entry); err != nil {
		return err
	}

	if isNew {
		// New unpublished articles never post to discord.
		if entry.IsPublished {
			u.postToDiscord(ctx, entry, newNewsMessage(entry.BodyMD, entry.Title))
		}

		return nil
	}

	// The article already exists, so look up its tracked discord message to
	// keep the channel in sync with the published state.
	current := Article{}
	if err := u.repository.GetNewsByID(ctx, int(entry.NewsID), &current); err != nil {
		return err
	}

	if !entry.IsPublished {
		u.unpublishFromDiscord(ctx, current)

		return nil
	}

	// Edits update the tracked discord message in place so the channel mirrors
	// the article. Rows without a tracked id (articles posted before tracking
	// existed, or whose original post failed) get a fresh message instead.
	if current.DiscordMessageID == 0 {
		u.postToDiscord(ctx, entry, editNewsMessages(entry.BodyMD, entry.Title))

		return nil
	}

	if u.logChannelID == "" {
		return nil
	}

	messageID := strconv.FormatInt(current.DiscordMessageID, 10)
	if errEdit := u.discord.EditMessage(u.logChannelID, messageID,
		editNewsMessages(entry.BodyMD, entry.Title)); errEdit != nil {
		slog.Error("Failed to edit news discord message",
			slog.Int("news_id", int(entry.NewsID)),
			slog.String("message_id", messageID),
			slog.String("error", errEdit.Error()))
	}

	return nil
}

// unpublishFromDiscord deletes the tracked discord message for an article
// that was unpublished, clearing the tracked id on success so a later
// republish posts a fresh message. On failure the id is kept so a retry can
// clean up.
func (u News) unpublishFromDiscord(ctx context.Context, current Article) {
	if current.DiscordMessageID == 0 {
		return
	}

	if errDelete := u.deleteTrackedDiscordMessage(ctx, current); errDelete != nil {
		return
	}

	if errClear := u.repository.SetDiscordMessageID(ctx, current.NewsID, 0); errClear != nil {
		slog.Error("Failed to clear news discord message id",
			slog.Int("news_id", int(current.NewsID)),
			slog.String("error", errClear.Error()))
	}
}

// deleteTrackedDiscordMessage deletes the discord message tracked for the
// article, if any. Failures are logged, not returned as fatal: an outage must
// not block site-side news changes.
func (u News) deleteTrackedDiscordMessage(_ context.Context, current Article) error {
	if u.logChannelID == "" || current.DiscordMessageID == 0 {
		return nil
	}

	messageID := strconv.FormatInt(current.DiscordMessageID, 10)
	if errDelete := u.discord.DeleteMessage(u.logChannelID, messageID); errDelete != nil {
		slog.Error("Failed to delete news discord message",
			slog.Int("news_id", int(current.NewsID)),
			slog.String("message_id", messageID),
			slog.String("error", errDelete.Error()))

		return errDelete
	}

	return nil
}

// postToDiscord posts the article to the log channel and tracks the created
// message id so later edits can update it in place. Discord failures are
// logged, not returned: an outage must not block site-side news changes.
func (u News) postToDiscord(ctx context.Context, entry *Article, message *discordgo.MessageSend) {
	if u.logChannelID == "" || message == nil {
		return
	}

	messageID, errSend := u.discord.SendWithID(u.logChannelID, message)
	if errSend != nil {
		slog.Error("Failed to post news discord message",
			slog.Int("news_id", int(entry.NewsID)),
			slog.String("error", errSend.Error()))

		return
	}

	if messageID == "" {
		return
	}

	storedID, errParse := strconv.ParseInt(messageID, 10, 64)
	if errParse != nil {
		slog.Error("Failed to parse news discord message id",
			slog.Int("news_id", int(entry.NewsID)),
			slog.String("message_id", messageID),
			slog.String("error", errParse.Error()))

		return
	}

	if errTrack := u.repository.SetDiscordMessageID(ctx, entry.NewsID, storedID); errTrack != nil {
		slog.Error("Failed to track news discord message id",
			slog.Int("news_id", int(entry.NewsID)),
			slog.String("message_id", messageID),
			slog.String("error", errTrack.Error()))
	}
}

func (u News) DropNewsArticle(ctx context.Context, newsID int32) error {
	current := Article{}
	if errFetch := u.repository.GetNewsByID(ctx, int(newsID), &current); errFetch == nil {
		// Failure is already logged by deleteTrackedDiscordMessage.
		_ = u.deleteTrackedDiscordMessage(ctx, current)
	}

	if err := u.repository.DropNewsArticle(ctx, newsID); err != nil {
		return err
	}

	slog.Info("Deleted news article", slog.Int("news_id", int(newsID)))

	return nil
}
