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

func findComicByTitlePrefix(
	ctx context.Context,
	repo domain.ComicRepository,
	normalized normalizedComic,
) (domain.Comic, bool, error) {
	if len(normalized.Titles) == 0 {
		return domain.Comic{}, false, nil
	}
	primary := normalized.Titles[0]
	words := strings.Fields(primary)
	if len(words) == 0 {
		return domain.Comic{}, false, nil
	}
	result, err := repo.List(ctx, domain.ComicListQuery{
		SearchTitle: words[0],
		Limit:       200,
	})
	if err != nil {
		return domain.Comic{}, false, err
	}
	var matches []domain.Comic
	for _, comic := range result.Comics {
		if comic.Deleted {
			continue
		}
		if normalized.ComType != ComTypeUnknown && comic.ComType != normalized.ComType {
			continue
		}
		for _, stored := range comic.Titles {
			if scrapeTitleMatchesStored(primary, stored) {
				matches = append(matches, comic)
				break
			}
		}
	}
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
