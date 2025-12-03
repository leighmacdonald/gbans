package wiki

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/parser"
	"github.com/leighmacdonald/gbans/internal/auth/permission"
	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/internal/notification"
	"github.com/microcosm-cc/bluemonday"
)

var ErrSlugUnknown = errors.New("slug unknown")

func Render(page Page) []byte {
	unsafeHTML := markdown.ToHTML([]byte(page.BodyMD), NewParser(), nil)

	return bluemonday.UGCPolicy().SanitizeBytes(unsafeHTML)
}

func NewParser() *parser.Parser {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.Tables

	return parser.NewWithExtensions(extensions)
}

// RootSlug is the top-most (index) page of the wiki.
const RootSlug = "home"

type Page struct {
	Slug            string               `json:"slug" binding:"required,gte=1,lte=64"`
	BodyMD          string               `json:"body_md" binding:"required,gte=1"`
	Revision        int                  `json:"revision" binding:"gte=0"`
	PermissionLevel permission.Privilege `json:"permission_level" binding:"required"`
	CreatedOn       time.Time            `json:"created_on"`
	UpdatedOn       time.Time            `json:"updated_on"`
}

func (page Page) NewRevision() Page {
	newPage := page
	newPage.UpdatedOn = time.Now()

	return newPage
}

func (page Page) Path() string {
	return "/wiki/" + page.Slug
}

func NewPage(slug string, body string) Page {
	now := time.Now()

	return Page{
		Slug:            slug,
		BodyMD:          body,
		PermissionLevel: permission.Guest,
		CreatedOn:       now,
		UpdatedOn:       now,
	}
}

type Wiki struct {
	Repository

	notif           notification.Notifier
	publicChannelID string
	channelID       string
}

func New(repository Repository, notif notification.Notifier, publicChannelID string, channelID string) Wiki {
	return Wiki{Repository: repository, notif: notif, publicChannelID: publicChannelID, channelID: channelID}
}

func (w *Wiki) Page(ctx context.Context, slug string) (Page, error) {
	slug = strings.ToLower(slug)
	if slug[0] == '/' {
		slug = slug[1:]
	}

	page, errGetWikiSlug := w.Repository.Page(ctx, slug)
	if errGetWikiSlug != nil {
		return page, errGetWikiSlug
	}

	return page, nil
}

func (w *Wiki) Delete(ctx context.Context, slug string) error {
	return w.Repository.Delete(ctx, slug)
}

func (w *Wiki) Save(ctx context.Context, update Page) (Page, error) {
	if update.Slug == "" || update.BodyMD == "" {
		return Page{}, httphelper.ErrInvalidParameter
	}

	var previous Page
	page, errGetWikiSlug := w.Page(ctx, update.Slug)
	if errGetWikiSlug != nil {
		if errors.Is(errGetWikiSlug, database.ErrNoResult) {
			page.CreatedOn = time.Now()
			page.Slug = update.Slug
		} else {
			return page, errGetWikiSlug
		}
		previous = page
	} else {
		page = page.NewRevision()
	}

	page.Revision++
	page.PermissionLevel = update.PermissionLevel
	page.BodyMD = update.BodyMD

	if errSave := w.Repository.Save(ctx, page); errSave != nil {
		page.Revision--

		return page, errSave
	}

	if page.Revision == 1 {
		go w.notif.Send(notification.NewDiscord(w.publicChannelID, pageCreated(page)))
	} else {
		go w.notif.Send(notification.NewDiscord(w.channelID, pageEdited(page, previous)))
	}

	return page, nil
}
