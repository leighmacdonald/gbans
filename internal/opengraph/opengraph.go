package opengraph

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"strings"
	"sync"
)

var ErrInvalidPattern = errors.New("invalid opengraph route pattern")

// Tag is a single meta tag rendered into the <head> of a response.
type Tag struct {
	// Property is the tag name, e.g. "og:title" or "twitter:card".
	Property string
	// Content is the tag value, html attribute escaped when rendered.
	Content string
}

// Tags defines the opengraph values for a single response.
type Tags struct {
	// Title is the page title. When set, the rendered value is "{title} - {site_name}".
	Title string
	// Description is the page description.
	Description string
	// Image is the social card image url. Relative urls are resolved against
	// the base url defined in Options.
	Image string
	// URL is the canonical url of the page. Relative urls are resolved against
	// the base url defined in Options.
	URL string
	// Type is the og:type value. Defaults to "website".
	Type string
	// Extra defines additional raw tags rendered after the standard ones.
	Extra []Tag
}

// TagsFunc resolves the tags for a single matched route. A returned error falls
// back to the site defaults and never fails the request.
type TagsFunc func(ctx context.Context, r *http.Request, params map[string]string) (Tags, error)

// Static returns a TagsFunc that always resolves to the given title and
// description. It is used for routes that do not depend on request data.
func Static(title string, description string) TagsFunc {
	return func(context.Context, *http.Request, map[string]string) (Tags, error) {
		return Tags{Title: title, Description: description}, nil
	}
}

// Options defines the site wide defaults used whenever resolved tag values are
// empty, as well as the base used to resolve relative urls.
type Options struct {
	SiteName        string
	SiteDescription string
	// BaseURL is the absolute public url of the site.
	BaseURL string
	// Image is the default social card image url.
	Image string
}

func (o *Options) normalize() {
	if o.Image == "" {
		o.Image = "/banner-default.webp"
	}

	if o.SiteName == "" {
		o.SiteName = "gbans"
	}
}

func NewRepository(opts Options) *Repository {
	repo := &Repository{opts: opts}
	repo.opts.normalize()

	return repo
}

// Repository is the central registration point for per route opengraph tags
// while also providing the middleware used to inject the resolved tags into
// html responses generated for known crawler bots.
type Repository struct {
	mu     sync.RWMutex
	opts   Options
	routes []route
}

type route struct {
	pattern  string
	segments []segment
	fn       TagsFunc
}

// Register adds a tags resolver for the given route pattern. The pattern syntax
// mirrors the frontend SPA routes using serve mux style wildcards, ex.
// /ban/{ban_id} or /wiki/{slug...}. Invalid patterns are discarded and logged.
func (rep *Repository) Register(pattern string, tagsFunc TagsFunc) {
	segments, errCompile := compilePattern(pattern)
	if errCompile != nil {
		slog.Error("Discarding invalid opengraph route pattern",
			slog.String("pattern", pattern), slog.String("error", errCompile.Error()))

		return
	}

	if tagsFunc == nil {
		slog.Error("Discarding opengraph registration missing a tags function", slog.String("pattern", pattern))

		return
	}

	rep.mu.Lock()
	defer rep.mu.Unlock()

	rep.routes = append(rep.routes, route{pattern: pattern, segments: segments, fn: tagsFunc})
}

// resolve finds the longest matching registered route for the request path.
func (rep *Repository) resolve(urlPath string) (TagsFunc, map[string]string, bool) {
	parts := splitPath(urlPath)

	// Index routes such as /forums/ and /wiki/ are the same client route as
	// their slashless parent, so normalize before matching.
	if len(parts) > 1 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}

	rep.mu.RLock()
	defer rep.mu.RUnlock()

	matched := false

	var best []segment

	var found map[string]string

	var bestFunc TagsFunc

	for _, routeEntry := range rep.routes {
		foundParams, ok := matchSegments(routeEntry.segments, parts)
		if !ok {
			continue
		}

		if !matched || len(routeEntry.segments) > len(best) {
			matched = true
			best = routeEntry.segments
			found = foundParams
			bestFunc = routeEntry.fn
		}
	}

	return bestFunc, found, matched
}

func splitPath(urlPath string) []string {
	clean := strings.Trim(urlPath, "/")
	if clean == "" {
		return []string{""}
	}

	return strings.Split(clean, "/")
}

// resolveTags merges the callback tags with the site wide defaults.
func (rep *Repository) resolveTags(ctx context.Context, r *http.Request, fn TagsFunc, params map[string]string) Tags {
	var tags Tags

	if fn != nil {
		resolved, errResolve := fn(ctx, r, params)
		if errResolve != nil {
			// Callback errors fall back to the site defaults since these pages
			// are not intended to fail for crawler bots.
			slog.Warn("Failed to resolve opengraph tags", slog.String("error", errResolve.Error()))
		} else {
			tags = resolved
		}
	}

	if tags.Title == "" {
		tags.Title = rep.opts.SiteName
	} else if tags.Title != rep.opts.SiteName {
		tags.Title = fmt.Sprintf("%s - %s", tags.Title, rep.opts.SiteName)
	}

	if tags.Description == "" {
		tags.Description = rep.opts.SiteDescription
	}

	tags.Image = rep.absoluteURL(tags.Image, rep.opts.Image)

	if tags.URL == "" {
		tags.URL = rep.opts.BaseURL + r.URL.Path
	} else {
		tags.URL = rep.absoluteURL(tags.URL, tags.URL)
	}

	if tags.Type == "" {
		tags.Type = "website"
	}

	return tags
}

// absoluteURL resolves a potentially relative url against the site base url.
func (rep *Repository) absoluteURL(url string, fallback string) string {
	if url == "" {
		return fallback
	}

	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}

	return rep.opts.BaseURL + url
}

// render generates the meta tags to inject into the document head.
func render(tags Tags, siteName string) string {
	var builder strings.Builder

	appendTag(&builder, "og:type", tags.Type)
	appendTag(&builder, "og:site_name", siteName)
	appendTag(&builder, "og:title", tags.Title)
	appendTag(&builder, "og:description", tags.Description)

	if tags.Image != "" {
		appendTag(&builder, "og:image", tags.Image)
		appendTag(&builder, "og:image:alt", siteName)
	}

	appendTag(&builder, "og:url", tags.URL)

	if tags.Image != "" {
		appendTag(&builder, "twitter:card", "summary_large_image")
		appendTag(&builder, "twitter:image", tags.Image)
	} else {
		appendTag(&builder, "twitter:card", "summary")
	}

	appendTag(&builder, "twitter:title", tags.Title)
	appendTag(&builder, "twitter:description", tags.Description)

	for _, tag := range tags.Extra {
		appendTag(&builder, tag.Property, tag.Content)
	}

	return builder.String()
}

// appendTag renders one <meta /> tag. OpenGraph tags use the property
// attribute, everything else, like the twitter ones, use name.
func appendTag(builder *strings.Builder, tagName string, content string) {
	if content == "" {
		return
	}

	attrName := "name"
	if strings.HasPrefix(tagName, "og:") {
		attrName = "property"
	}

	builder.WriteString("        <meta ")
	builder.WriteString(attrName)
	builder.WriteString(`="`)
	builder.WriteString(html.EscapeString(tagName))
	builder.WriteString(`" content="`)
	builder.WriteString(html.EscapeString(content))
	builder.WriteString("\" />\n")
}

// inject splices the meta tags into the html body before the </head> tag,
// appending to the body when no head tag exists.
func inject(body string, tagsHTML string) string {
	headClose := "</head>"
	rootOpen := "<html"

	lower := strings.ToLower(body)
	if idx := strings.Index(lower, headClose); idx != -1 {
		return body[:idx] + tagsHTML + body[idx:]
	}

	if idx := strings.Index(lower, rootOpen); idx != -1 {
		if closeIdx := strings.Index(body[idx:], ">"); closeIdx != -1 {
			insertAt := idx + closeIdx + 1

			return body[:insertAt] + tagsHTML + body[insertAt:]
		}
	}

	return tagsHTML + body
}
