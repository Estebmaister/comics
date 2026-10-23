package discovery

import (
	"context"
	"strings"

	"comics/domain"
	"comics/internal/identity"
)

type MatchKind string

const (
	MatchNone         MatchKind = ""
	MatchDiscoveryKey MatchKind = "discovery_key"
	MatchTitlePrefix  MatchKind = "title_prefix"
	MatchTitleTokens  MatchKind = "title_tokens"
)

var titleLookupStopWords = map[string]struct{}{
	"became": {}, "become": {}, "being": {}, "return": {}, "returns": {},
	"from": {}, "with": {}, "into": {}, "that": {}, "this": {},
}

// ResolveComic finds an existing comic for a scraped or created title.
func ResolveComic(
	ctx context.Context,
	repo domain.ComicRepository,
	incomingTitle string,
	comType int,
) (domain.Comic, MatchKind, bool, error) {
	titles := identity.NormalizeTitleVariants([]string{incomingTitle}, comType)
	if len(titles) == 0 {
		return domain.Comic{}, MatchNone, false, nil
	}
	primary := titles[0]
	scope := identity.DiscoveryScope(primary, comType)
	titleKeys := identity.DiscoveryTitleKeysForTitles(titles, comType)

	matches, err := repo.FindComicsByDiscoveryTitleKeys(ctx, scope, titleKeys)
	if err != nil {
		return domain.Comic{}, MatchNone, false, err
	}
	if comic, ok := pickBestMatch(matches, primary, comType); ok {
		return comic, MatchDiscoveryKey, true, nil
	}

	token := titleLookupSearchTerm(primary)
	if token != "" {
		candidates, err := repo.FindComicsByDiscoveryKeyToken(ctx, scope, token, 200)
		if err != nil {
			return domain.Comic{}, MatchNone, false, err
		}
		matches = filterByTitleRelation(candidates, primary, comType)
		if comic, ok := pickBestMatch(matches, primary, comType); ok {
			kind := MatchTitlePrefix
			if matchUsesTokens(primary, comic) {
				kind = MatchTitleTokens
			}
			return comic, kind, true, nil
		}
	}

	return domain.Comic{}, MatchNone, false, nil
}

func matchUsesTokens(primary string, comic domain.Comic) bool {
	for _, stored := range comic.Titles {
		if identity.TitlesMatchBySignificantTokens(primary, stored) {
			return true
		}
	}
	return false
}

func filterByTitleRelation(candidates []domain.Comic, primary string, comType int) []domain.Comic {
	out := make([]domain.Comic, 0, len(candidates))
	for _, comic := range candidates {
		if !comicInLookupScope(comic, comType) {
			continue
		}
		for _, stored := range comic.Titles {
			if identity.TitleMatchKey(stored) == identity.TitleMatchKey(primary) ||
				identity.TitlesArePrefixMatch(primary, stored) ||
				identity.TitlesMatchBySignificantTokens(primary, stored) {
				out = append(out, comic)
				break
			}
		}
	}
	return out
}

func comicInLookupScope(comic domain.Comic, incomingComType int) bool {
	comicIsNovel := comic.ComType == identity.NovelType
	for _, stored := range comic.Titles {
		if identity.IsNovelIdentity(stored, comic.ComType) {
			comicIsNovel = true
			break
		}
	}
	incomingIsNovel := incomingComType == identity.NovelType
	return comicIsNovel == incomingIsNovel
}

func titleLookupSearchTerm(primaryTitle string) string {
	words := strings.Fields(strings.TrimSpace(primaryTitle))
	if len(words) == 0 {
		return ""
	}
	var candidates []string
	for _, word := range words {
		lower := strings.ToLower(word)
		if len(word) < 4 {
			continue
		}
		if lower == "the" || lower == "a" || lower == "an" {
			continue
		}
		if _, stop := titleLookupStopWords[lower]; stop {
			continue
		}
		candidates = append(candidates, lower)
	}
	if len(candidates) == 0 {
		for _, word := range words {
			lower := strings.ToLower(word)
			if lower != "the" && lower != "a" && lower != "an" && word != "" {
				return lower
			}
		}
		return strings.ToLower(words[0])
	}
	best := candidates[0]
	for _, word := range candidates[1:] {
		if len(word) > len(best) {
			best = word
		} else if len(word) == len(best) {
			best = word
		}
	}
	return best
}

func pickBestMatch(matches []domain.Comic, incomingPrimary string, comType int) (domain.Comic, bool) {
	if len(matches) == 0 {
		return domain.Comic{}, false
	}
	incomingKey := identity.TitleMatchKey(incomingPrimary)
	best := matches[0]
	bestScore := -1
	for _, comic := range matches {
		if !comicInLookupScope(comic, comType) {
			continue
		}
		score := 0
		for _, stored := range comic.Titles {
			if identity.TitleMatchKey(stored) == incomingKey {
				score += 10
				if strings.EqualFold(strings.TrimSpace(stored), strings.TrimSpace(incomingPrimary)) {
					score += 5
				}
			} else if identity.TitlesMatchBySignificantTokens(incomingPrimary, stored) {
				score += 8
			} else if identity.TitlesArePrefixMatch(incomingPrimary, stored) {
				score += 6
			}
		}
		if comic.Track {
			score += 2
		}
		score += comic.CurrentChap / 100
		if score > bestScore || (score == bestScore && comic.ID < best.ID) {
			bestScore = score
			best = comic
		}
	}
	if bestScore < 6 {
		return domain.Comic{}, false
	}
	return best, true
}
