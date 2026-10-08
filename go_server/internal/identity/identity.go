// Package identity mirrors Python src/db/identity.py key rules for comic dedupe.
package identity

import (
	"regexp"
	"strings"
	"unicode"
)

const (
	NovelType            = 4
	NovelSuffix          = " - novel"
	NovelIdentityPrefix  = "novel:"
	SeriesIdentityPrefix = "series:"
)

var (
	novelMarkerRe    = regexp.MustCompile(`(?i)\(\s*novel\s*\)|-\s*novel\s*$`)
	leadingArticleRe = regexp.MustCompile(`(?i)^(the|a|an)\s+`)
	nonMatchCharRe   = regexp.MustCompile(`[^a-z0-9]+`)
)

func SplitTitleValues(titles string) []string {
	if strings.TrimSpace(titles) == "" {
		return nil
	}
	parts := strings.Split(titles, "|")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, part)
	}
	return out
}

func NormalizeText(value string) string {
	value = strings.ReplaceAll(value, "\u2019", "'")
	value = strings.ReplaceAll(value, "\u2018", "'")
	value = strings.TrimSpace(value)
	return value
}

func StripNovelMarker(title string) string {
	normalized := NormalizeText(title)
	if normalized == "" {
		return ""
	}
	stripped := novelMarkerRe.ReplaceAllString(normalized, "")
	return NormalizeText(stripped)
}

func TitleHasNovelMarker(title string) bool {
	return novelMarkerRe.MatchString(NormalizeText(title))
}

func IsNovelIdentity(title string, comType int) bool {
	return comType == NovelType || TitleHasNovelMarker(title)
}

func NormalizeTitleForStorage(title string, comType int, primary bool) string {
	cleaned := StripNovelMarker(title)
	if cleaned == "" {
		return ""
	}
	if primary && IsNovelIdentity(title, comType) {
		cleaned += NovelSuffix
	}
	return sentenceCase(cleaned)
}

func sentenceCase(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	runes := []rune(strings.ToLower(value))
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func NormalizeTitleVariants(titles []string, comType int) []string {
	normalized := make([]string, 0, len(titles))
	seen := map[string]struct{}{}
	for index, rawTitle := range titles {
		title := NormalizeTitleForStorage(rawTitle, comType, index == 0)
		if title == "" {
			continue
		}
		key := strings.ToLower(title)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, title)
	}
	return normalized
}

func PrimaryTitleFromTitles(titles []string) string {
	for _, title := range titles {
		normalized := NormalizeText(title)
		if normalized != "" {
			return normalized
		}
	}
	return ""
}

func NormalizePrimaryTitle(title string) string {
	return strings.ToLower(StripNovelMarker(title))
}

func TitleMatchKey(title string) string {
	normalized := NormalizePrimaryTitle(title)
	normalized = leadingArticleRe.ReplaceAllString(normalized, "")
	return nonMatchCharRe.ReplaceAllString(normalized, "")
}

// TitlesArePrefixMatch reports whether one normalized title key is a prefix of the
// other (truncated publisher listings vs full title). Mirrors Python identity.py.
func TitlesArePrefixMatch(incomingTitle string, storedTitle string) bool {
	incomingKey := TitleMatchKey(incomingTitle)
	storedKey := TitleMatchKey(storedTitle)
	if incomingKey == "" || storedKey == "" {
		return false
	}
	return strings.HasPrefix(storedKey, incomingKey) || strings.HasPrefix(incomingKey, storedKey)
}

func BuildIdentityKey(primaryTitle string, comType int) string {
	normalizedTitle := NormalizePrimaryTitle(primaryTitle)
	if normalizedTitle == "" {
		return ""
	}
	prefix := SeriesIdentityPrefix
	if IsNovelIdentity(primaryTitle, comType) {
		prefix = NovelIdentityPrefix
	}
	return prefix + normalizedTitle
}

func BuildIdentityKeyFromTitles(titles []string, comType int) string {
	return BuildIdentityKey(PrimaryTitleFromTitles(titles), comType)
}
