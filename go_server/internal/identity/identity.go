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
	novelMarkerRe          = regexp.MustCompile(`(?i)\(\s*novel\s*\)|-\s*novel\s*$`)
	leadingArticleRe       = regexp.MustCompile(`(?i)^(the|a|an)\s+`)
	nonMatchCharRe         = regexp.MustCompile(`[^a-z0-9]+`)
	nonUnicodeMatchCharRe  = regexp.MustCompile(`[^\p{L}\p{N}]+`)
)

// ScopedTitleKey is one discovery index row (scope + normalized title_key).
type ScopedTitleKey struct {
	Scope    string `json:"scope"`
	TitleKey string `json:"title_key"`
}

// significantTokenStopWords filters tokens that are too common to distinguish series.
var significantTokenStopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "at": {}, "became": {}, "become": {}, "being": {},
	"for": {}, "from": {}, "i": {}, "in": {}, "into": {}, "my": {}, "of": {}, "on": {},
	"or": {}, "that": {}, "the": {}, "this": {}, "to": {}, "with": {},
}

func SplitTitleValues(titles string) []string {
	if strings.TrimSpace(titles) == "" {
		return nil
	}
	parts := strings.Split(titles, "|")
	out := make([]string, 0, len(parts))
	out = append(out, parts...)
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
	asciiKey := nonMatchCharRe.ReplaceAllString(normalized, "")
	if asciiKey != "" {
		return asciiKey
	}
	return nonUnicodeMatchCharRe.ReplaceAllString(normalized, "")
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

func normalizeSignificantToken(word string) string {
	word = strings.ToLower(strings.TrimSpace(word))
	if word == "" {
		return ""
	}
	if len(word) > 4 && strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss") {
		word = strings.TrimSuffix(word, "s")
	}
	return word
}

// TitleSignificantTokenSet extracts discriminating words from a title for fuzzy matching
// (reordered English titles, e.g. "Return of the frozen player" vs "The frozen player returns").
func TitleSignificantTokenSet(title string) map[string]struct{} {
	cleaned := strings.ToLower(StripNovelMarker(title))
	parts := nonMatchCharRe.Split(cleaned, -1)
	tokens := map[string]struct{}{}
	for _, part := range parts {
		token := normalizeSignificantToken(part)
		if len(token) < 3 {
			continue
		}
		if _, stop := significantTokenStopWords[token]; stop {
			continue
		}
		tokens[token] = struct{}{}
	}
	return tokens
}

// TitlesMatchBySignificantTokens is true when every token from the smaller title appears
// in the larger title (subset match), with at least two shared tokens when possible.
func TitlesMatchBySignificantTokens(incomingTitle string, storedTitle string) bool {
	incoming := TitleSignificantTokenSet(incomingTitle)
	stored := TitleSignificantTokenSet(storedTitle)
	if len(incoming) == 0 || len(stored) == 0 {
		return false
	}
	smaller, larger := incoming, stored
	if len(stored) < len(incoming) {
		smaller, larger = stored, incoming
	}
	if len(smaller) == 1 {
		token := ""
		for t := range smaller {
			token = t
		}
		_, ok := larger[token]
		return ok
	}
	shared := 0
	for token := range smaller {
		if _, ok := larger[token]; ok {
			shared++
		}
	}
	if shared < len(smaller) {
		return false
	}
	if len(smaller) >= 2 {
		return true
	}
	return shared >= 1
}

// DiscoveryScope is the dedupe namespace for a title variant (comic vs novel).
func DiscoveryScope(title string, comType int) string {
	if IsNovelIdentity(title, comType) {
		return strings.TrimSuffix(NovelIdentityPrefix, ":")
	}
	return strings.TrimSuffix(SeriesIdentityPrefix, ":")
}

// DiscoveryTitleKey is the normalized, article-stripped key stored in comic_title_keys.
func DiscoveryTitleKey(title string) string {
	return TitleMatchKey(title)
}

// DiscoveryKey is scope:titleKey (e.g. series:executioner).
func DiscoveryKey(title string, comType int) string {
	titleKey := DiscoveryTitleKey(title)
	if titleKey == "" {
		return ""
	}
	return DiscoveryScope(title, comType) + ":" + titleKey
}

// DiscoveryTitleKeyPairsForTitles returns unique scoped title_key pairs for all variants.
func DiscoveryTitleKeyPairsForTitles(titles []string, comType int) []ScopedTitleKey {
	seen := map[ScopedTitleKey]struct{}{}
	out := make([]ScopedTitleKey, 0, len(titles))
	for _, title := range titles {
		item := ScopedTitleKey{
			Scope:    DiscoveryScope(title, comType),
			TitleKey: DiscoveryTitleKey(title),
		}
		if item.TitleKey == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

// DiscoveryKeysForTitles returns unique discovery keys for all title variants.
func DiscoveryKeysForTitles(titles []string, comType int) []string {
	out := make([]string, 0, len(titles))
	seen := map[string]struct{}{}
	for _, pair := range DiscoveryTitleKeyPairsForTitles(titles, comType) {
		key := pair.Scope + ":" + pair.TitleKey
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

// DiscoveryTitleKeysForTitles returns unique title_key values for all variants in a scope.
func DiscoveryTitleKeysForTitles(titles []string, comType int) []string {
	out := make([]string, 0, len(titles))
	seen := map[string]struct{}{}
	for _, pair := range DiscoveryTitleKeyPairsForTitles(titles, comType) {
		scoped := pair.Scope + ":" + pair.TitleKey
		if _, ok := seen[scoped]; ok {
			continue
		}
		seen[scoped] = struct{}{}
		out = append(out, pair.TitleKey)
	}
	return out
}

// BuildIdentityKey is deprecated; use DiscoveryKey.
func BuildIdentityKey(primaryTitle string, comType int) string {
	return DiscoveryKey(primaryTitle, comType)
}

// BuildIdentityKeyFromTitles is deprecated; use DiscoveryKeysForTitles.
func BuildIdentityKeyFromTitles(titles []string, comType int) string {
	if len(titles) == 0 {
		return ""
	}
	return DiscoveryKey(PrimaryTitleFromTitles(titles), comType)
}
