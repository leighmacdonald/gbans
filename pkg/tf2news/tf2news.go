package tf2news

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/mmcdole/gofeed"
)

const (
	feedURL     = "https://www.teamfortress.com/rss.xml"
	updateTitle = "Team Fortress 2 Update Released"
)

var (
	ErrFetch = errors.New("failed to fetch news")
)

type FeedItem struct {
	Title       string
	Description string
	Link        string
	PublishedAt time.Time
	GameUpdate  bool
}

func Fetch(ctx context.Context) ([]*FeedItem, error) {
	client := &http.Client{Timeout: time.Second * 30}
	req, errReq := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if errReq != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetch, errReq)
	}
	resp, errResp := client.Do(req)
	if errResp != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetch, errResp)
	}
	defer resp.Body.Close()

	return Parse(ctx, resp.Body)
}

func Parse(ctx context.Context, body io.Reader) ([]*FeedItem, error) {
	parser := gofeed.NewParser()
	feed, err := parser.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFetch, err)
	}

	items := make([]*FeedItem, len(feed.Items))
	for i, item := range feed.Items {
		markdown, errConv := htmltomarkdown.ConvertString(item.Description)
		if errConv != nil {
			return items, fmt.Errorf("%w: %w", ErrFetch, errConv)
		}
		if item.PublishedParsed == nil {
			continue
		}
		items[i] = &FeedItem{
			Title:       item.Title,
			Description: markdown,
			Link:        item.Link,
			PublishedAt: *item.PublishedParsed,
			GameUpdate:  item.Title == updateTitle,
		}
	}

	return items, nil
}
