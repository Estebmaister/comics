package scrape

import (
	"regexp"
	"strconv"
	"strings"

	"comics/internal/identity"
)

const minimumCoverURLLength = 10

var chapterDigitsRe = regexp.MustCompile(`\d+`)

func parseChapterNumber(chapter string) (int, bool) {
	text := strings.TrimSpace(chapter)
	if text == "" {
		return 0, false
	}
	if value, err := strconv.Atoi(text); err == nil {
		return value, true
	}
	matches := chapterDigitsRe.FindAllString(text, -1)
	if len(matches) == 0 {
		return 0, false
	}
	value, err := strconv.Atoi(matches[len(matches)-1])
	if err != nil {
		return 0, false
	}
	return value, true
}

func parseComType(raw string) int {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "manga":
		return ComTypeManga
	case "manhua":
		return ComTypeManhua
	case "manhwa", "webtoon":
		return ComTypeManhwa
	case "novel":
		return ComTypeNovel
	default:
		return ComTypeUnknown
	}
}

func parseStatus(raw string) int {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "completed":
		return StatusCompleted
	case "ongoing":
		return StatusOnAir
	case "hiatus", "season end":
		return StatusBreak
	case "dropped":
		return StatusDropped
	default:
		return StatusUnknown
	}
}

func parseCoverURL(cover string, publisherID int) string {
	if strings.Contains(cover, "http") {
		index := strings.Index(cover, "http")
		return cover[index:]
	}
	if strings.HasPrefix(cover, "/") {
		name, err := publisherNameForID(publisherID)
		if err != nil {
			return ""
		}
		return publisherBaseURL(name) + cover
	}
	if len(cover) < minimumCoverURLLength {
		return ""
	}
	return cover
}

func normalizeScrapedComic(raw ScrapedComic, publisherID int) (normalizedComic, bool) {
	chapter, ok := parseChapterNumber(raw.Chapter)
	if !ok {
		return normalizedComic{}, false
	}
	title := identity.NormalizeText(strings.ReplaceAll(raw.Title, "...", ""))
	if title == "" {
		return normalizedComic{}, false
	}
	comType := parseComType(raw.ComType)
	titles := identity.NormalizeTitleVariants([]string{title}, comType)
	if len(titles) == 0 {
		return normalizedComic{}, false
	}
	return normalizedComic{
		Titles:      titles,
		CurrentChap: chapter,
		Cover:       parseCoverURL(raw.CoverURL, publisherID),
		ComType:     comType,
		Status:      parseStatus(raw.Status),
		Author:      identity.NormalizeText(raw.Author),
		Publisher:   publisherID,
		IdentityKey: identity.BuildIdentityKeyFromTitles(titles, comType),
	}, true
}

type normalizedComic struct {
	Titles      []string
	CurrentChap int
	Cover       string
	ComType     int
	Status      int
	Author      string
	Publisher   int
	IdentityKey string
}
