package scrape

import (
	"context"
	"strings"

	"comics/domain"
	"comics/internal/identity"
)

// scrapeTitleMatchesStored is true when a truncated scrape title matches a longer DB title.
func scrapeTitleMatchesStored(scrapedTitle string, storedTitle string) bool {
	incomingKey := identity.TitleMatchKey(scrapedTitle)
	storedKey := identity.TitleMatchKey(storedTitle)
	return incomingKey != "" && storedKey != "" && strings.HasPrefix(storedKey, incomingKey)
}

func comicInLookupScope(comic domain.Comic, incomingComType int) bool {
	comicIsNovel := comic.ComType == ComTypeNovel
	for _, stored := range comic.Titles {
		if identity.IsNovelIdentity(stored, comic.ComType) {
			comicIsNovel = true
			break
		}
	}
	incomingIsNovel := incomingComType == ComTypeNovel
	return comicIsNovel == incomingIsNovel
}

func collectTitleLookupMatches(
	result domain.ComicListResult,
	normalized normalizedComic,
	match func(primary string, stored string) bool,
) []domain.Comic {
	if len(normalized.Titles) == 0 {
		return nil
	}
	primary := normalized.Titles[0]
	var matches []domain.Comic
	for _, comic := range result.Comics {
		if comic.Deleted {
			continue
		}
		if !comicInLookupScope(comic, normalized.ComType) {
			continue
		}
		for _, stored := range comic.Titles {
			if match(primary, stored) {
				matches = append(matches, comic)
				break
			}
		}
	}
	return matches
}

func listTitleLookupCandidates(
	ctx context.Context,
	repo domain.ComicRepository,
	normalized normalizedComic,
) (domain.ComicListResult, error) {
	if len(normalized.Titles) == 0 {
		return domain.ComicListResult{}, nil
	}
	words := strings.Fields(normalized.Titles[0])
	if len(words) == 0 {
		return domain.ComicListResult{}, nil
	}
	return repo.List(ctx, domain.ComicListQuery{
		SearchTitle: words[0],
		Limit:       200,
	})
}

func findComicByExactTitleKey(
	ctx context.Context,
	repo domain.ComicRepository,
	normalized normalizedComic,
) (domain.Comic, bool, error) {
	result, err := listTitleLookupCandidates(ctx, repo, normalized)
	if err != nil {
		return domain.Comic{}, false, err
	}
	incomingKey := identity.TitleMatchKey(normalized.Titles[0])
	if incomingKey == "" {
		return domain.Comic{}, false, nil
	}
	matches := collectTitleLookupMatches(
		result,
		normalized,
		func(_ string, stored string) bool {
			return identity.TitleMatchKey(stored) == incomingKey
		},
	)
	if len(matches) != 1 {
		return domain.Comic{}, false, nil
	}
	return matches[0], true, nil
}

func findComicByTitlePrefix(
	ctx context.Context,
	repo domain.ComicRepository,
	normalized normalizedComic,
) (domain.Comic, bool, error) {
	result, err := listTitleLookupCandidates(ctx, repo, normalized)
	if err != nil {
		return domain.Comic{}, false, err
	}
	matches := collectTitleLookupMatches(
		result,
		normalized,
		scrapeTitleMatchesStored,
	)
	if len(matches) != 1 {
		return domain.Comic{}, false, nil
	}
	return matches[0], true, nil
}

func mergeTitleVariants(existing []string, incoming []string, comType int) []string {
	seen := map[string]struct{}{}
	combined := make([]string, 0, len(existing)+len(incoming))
	for _, title := range append(existing, incoming...) {
		key := strings.ToLower(strings.TrimSpace(title))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		combined = append(combined, title)
	}
	return identity.NormalizeTitleVariants(combined, comType)
}
