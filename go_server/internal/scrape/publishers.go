package scrape

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var chapterNameDigitsRe = regexp.MustCompile(`\d`)
var flameChapterLineRe = regexp.MustCompile(`(?i)Chapter\s+(\d+)`)
var flameCoverEncodedRe = regexp.MustCompile(`uploads%2Fimages%2Fseries%2F\d+%2F[^&"]+`)

func loadDocument(ctx context.Context, url string, fetcher PageFetcher) (*goquery.Document, error) {
	if fetcher == nil {
		fetcher = NewHTTPFetcher()
	}
	html, err := fetcher.FetchHTML(ctx, url)
	if err != nil {
		return nil, err
	}
	return goquery.NewDocumentFromReader(strings.NewReader(html))
}

func scrapeAsuraPage(
	ctx context.Context,
	url string,
	publisherID int,
	fetcher PageFetcher,
	register func(ScrapedComic, int) error,
) error {
	doc, err := loadDocument(ctx, url, fetcher)
	if err != nil {
		return err
	}
	seen := map[string]struct{}{}
	doc.Find("astro-island[props]").Each(func(_ int, sel *goquery.Selection) {
		rawProps, ok := sel.Attr("props")
		if !ok {
			return
		}
		for _, comic := range extractAsuraComics(rawProps) {
			key := strings.ToLower(comic.Title)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			if err := register(comic, publisherID); err != nil {
				slog.Warn("scrape register failed",
					"publisher", "Asura",
					"title", comic.Title,
					"error", err,
				)
			}
		}
	})
	return nil
}

func extractAsuraComics(rawProps string) []ScrapedComic {
	props, err := parseAstroIslandProps(rawProps)
	if err != nil {
		return nil
	}
	comics := make([]ScrapedComic, 0)
	for _, key := range []string{"chapters", "items", "initialSeries"} {
		collection, ok := props[key].([]any)
		if !ok {
			continue
		}
		for _, entry := range collection {
			item, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			if comic, ok := comicFromAsuraEntry(item); ok {
				comics = append(comics, comic)
			}
		}
	}
	return comics
}

func comicFromAsuraEntry(entry map[string]any) (ScrapedComic, bool) {
	title := firstString(entry, "comic_name", "title")
	cover := firstString(entry, "comic_cover", "cover_url", "cover")
	comType := firstString(entry, "type")
	if comType == "" {
		comType = "manhwa"
	}
	status := firstString(entry, "status")
	if status == "" {
		status = "ongoing"
	}
	chapter := firstAnyString(entry, "number", "chapter_count")
	if chapter == "" {
		if name := firstString(entry, "name"); name != "" && chapterNameDigitsRe.MatchString(name) {
			chapter = name
		}
	}
	if title == "" || chapter == "" {
		return ScrapedComic{}, false
	}
	return ScrapedComic{
		Chapter:  chapter,
		Title:    title,
		CoverURL: cover,
		ComType:  comType,
		Status:   status,
	}, true
}

func parseAstroIslandProps(rawProps string) (map[string]any, error) {
	var parsed any
	if err := json.Unmarshal([]byte(rawProps), &parsed); err != nil {
		return nil, err
	}
	decoded := astroDecodeObject(parsed)
	result, ok := decoded.(map[string]any)
	if !ok {
		return map[string]any{}, nil
	}
	return result, nil
}

func astroDecode(value any) any {
	items, ok := value.([]any)
	if !ok || len(items) != 2 {
		return value
	}
	typeID, ok := toInt(items[0])
	if !ok {
		return value
	}
	data := items[1]
	switch typeID {
	case 0:
		return astroDecodeObject(data)
	case 1:
		list, ok := data.([]any)
		if !ok {
			return data
		}
		out := make([]any, len(list))
		for i, item := range list {
			out[i] = astroDecode(item)
		}
		return out
	case 4:
		obj, ok := data.(map[string]any)
		if !ok {
			return data
		}
		out := make(map[string]any, len(obj))
		for key, val := range obj {
			out[key] = astroDecode(val)
		}
		return out
	case 5:
		list, ok := data.([]any)
		if !ok {
			return data
		}
		out := make([]any, len(list))
		for i, item := range list {
			out[i] = astroDecode(item)
		}
		return out
	default:
		return data
	}
}

func astroDecodeObject(data any) any {
	obj, ok := data.(map[string]any)
	if !ok {
		return data
	}
	out := make(map[string]any, len(obj))
	for key, val := range obj {
		out[key] = astroDecode(val)
	}
	return out
}

func firstString(entry map[string]any, keys ...string) string {
	return firstAnyString(entry, keys...)
}

func firstAnyString(entry map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := entry[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			if typed != "" {
				return typed
			}
		case float64:
			return strconv.FormatInt(int64(typed), 10)
		case int:
			return strconv.Itoa(typed)
		}
	}
	return ""
}

func toInt(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	default:
		return 0, false
	}
}

func scrapeManhuaPlusPage(
	ctx context.Context,
	url string,
	publisherID int,
	fetcher PageFetcher,
	register func(ScrapedComic, int) error,
) error {
	doc, err := loadDocument(ctx, url, fetcher)
	if err != nil {
		return err
	}
	doc.Find("div.page-item-detail").Each(func(_ int, comicDiv *goquery.Selection) {
		info := comicDiv.Find("div.item-summary").First()
		title := strings.TrimSpace(info.Find("div.post-title h3 a").First().Text())
		chapter := strings.TrimSpace(info.Find("div.chapter-item").First().Find("span a").First().Text())
		cover, _ := comicDiv.Find("div.a img").First().Attr("data-src")
		if title == "" || chapter == "" {
			return
		}
		_ = register(ScrapedComic{
			Chapter:  chapter,
			Title:    title,
			CoverURL: cover,
			ComType:  "manhua",
			Status:   "ongoing",
		}, publisherID)
	})
	return nil
}

func scrapeFlamePage(
	ctx context.Context,
	url string,
	publisherID int,
	fetcher PageFetcher,
	register func(ScrapedComic, int) error,
) error {
	doc, err := loadDocument(ctx, url, fetcher)
	if err != nil {
		return err
	}
	doc.Find("[class*='chapterCardContainer']").Each(func(_ int, card *goquery.Selection) {
		link := card.Find("a[class*='chapterImageLink']").First()
		if link.Length() == 0 {
			return
		}
		title := strings.TrimSpace(link.AttrOr("title", ""))
		if title == "" {
			title = strings.TrimSpace(link.Find("img").AttrOr("alt", ""))
		}
		cardHTML, err := card.Html()
		if err != nil || title == "" {
			return
		}
		chapterMatch := flameChapterLineRe.FindStringSubmatch(cardHTML)
		if len(chapterMatch) < 2 {
			return
		}
		status := "ongoing"
		cardText := strings.ToLower(card.Text())
		if strings.Contains(cardText, "completed") {
			status = "completed"
		}
		_ = register(ScrapedComic{
			Chapter:  chapterMatch[1],
			Title:    title,
			CoverURL: parseFlameCoverURL(cardHTML),
			ComType:  "manhwa",
			Status:   status,
		}, publisherID)
	})
	return nil
}

func parseFlameCoverURL(cardHTML string) string {
	encoded := flameCoverEncodedRe.FindString(cardHTML)
	if encoded == "" {
		return ""
	}
	path := strings.ReplaceAll(encoded, "%2F", "/")
	path = strings.ReplaceAll(path, "%3F", "?")
	return "https://flamecomics.xyz/" + path
}

func scrapeRealmPage(
	ctx context.Context,
	url string,
	publisherID int,
	fetcher PageFetcher,
	register func(ScrapedComic, int) error,
) error {
	doc, err := loadDocument(ctx, url, fetcher)
	if err != nil {
		return err
	}
	doc.Find("div.uta").Each(func(_ int, comicDiv *goquery.Selection) {
		link := comicDiv.Find("div.a").First()
		cover, _ := link.Find("img").First().Attr("src")
		status := strings.TrimSpace(link.Find("div span").First().Text())
		info := comicDiv.Find("div.luf").First()
		title := strings.TrimSpace(info.Find("a h4").First().Text())
		chapterList := info.Find("ul").First()
		if chapterList.Length() == 0 {
			return
		}
		chapter := strings.TrimSpace(chapterList.Find("li a").First().Text())
		comType := "manhwa"
		if classes, ok := chapterList.Attr("class"); ok {
			parts := strings.Fields(classes)
			if len(parts) > 0 {
				comType = parts[0]
			}
		}
		if title == "" || chapter == "" {
			return
		}
		_ = register(ScrapedComic{
			Chapter:  chapter,
			Title:    title,
			CoverURL: cover,
			ComType:  comType,
			Status:   status,
		}, publisherID)
	})
	return nil
}

func scrapeDemonicPage(
	ctx context.Context,
	url string,
	publisherID int,
	fetcher PageFetcher,
	register func(ScrapedComic, int) error,
) error {
	doc, err := loadDocument(ctx, url, fetcher)
	if err != nil {
		return err
	}
	doc.Find("div.updates-element.border-box").Each(func(_ int, comicDiv *goquery.Selection) {
		cover, _ := comicDiv.Find("div.thumb a img").First().Attr("src")
		comicInt := comicDiv.Find("div.updates-element-info").First()
		if comicInt.Length() == 0 {
			comicInt = comicDiv.Find("div.flex-row").Children().Eq(1)
		}
		titleLink := comicInt.Find("h2 a").First()
		title := strings.TrimSpace(titleLink.Text())
		if title == "" {
			title = strings.TrimSpace(titleLink.AttrOr("title", ""))
		}
		title = strings.ReplaceAll(title, "...", "")
		comType := "manhwa"
		if _, hasStyle := comicDiv.Attr("style"); hasStyle {
			title += " - novel"
			comType = "novel"
		}
		chapter := strings.TrimSpace(comicInt.Find("a.chplinks").First().Text())
		if chapter == "" {
			links := comicInt.Find("a")
			if links.Length() >= 2 {
				chapter = strings.TrimSpace(links.Eq(1).Text())
			}
		}
		if title == "" || chapter == "" {
			return
		}
		_ = register(ScrapedComic{
			Chapter:  chapter,
			Title:    title,
			CoverURL: cover,
			ComType:  comType,
			Status:   "ongoing",
		}, publisherID)
	})
	return nil
}

func scrapeManganatoPage(
	ctx context.Context,
	url string,
	publisherID int,
	fetcher PageFetcher,
	register func(ScrapedComic, int) error,
) error {
	doc, err := loadDocument(ctx, url, fetcher)
	if err != nil {
		return err
	}
	doc.Find("div.itemupdate.first").Each(func(_ int, comicDiv *goquery.Selection) {
		cover, _ := comicDiv.Find("a img").First().Attr("src")
		comicInt := comicDiv.Find("ul").First()
		title := strings.TrimSpace(comicInt.Find("li h3 a").First().Text())
		author := strings.TrimSpace(comicInt.Find("li span").First().Text())
		items := comicInt.Find("li")
		if items.Length() < 2 {
			return
		}
		chapter := strings.TrimSpace(items.Eq(1).Find("span a").First().Text())
		if title == "" || chapter == "" {
			return
		}
		_ = register(ScrapedComic{
			Chapter:  chapter,
			Title:    title,
			CoverURL: cover,
			ComType:  "manhwa",
			Status:   "ongoing",
			Author:   author,
		}, publisherID)
	})
	return nil
}
