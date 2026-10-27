package discovery

import (
	"context"

	"comics/domain"
	"comics/internal/identity"
)

// FilterTitlesForMerge drops title variants whose discovery key is owned by a third
// active comic (not the merge base or duplicate). Prevents merge from failing when
// the base row still carries an alternate title that belongs to another series.
func FilterTitlesForMerge(
	ctx context.Context,
	repo domain.ComicRepository,
	baseID int,
	mergingID int,
	comType int,
	titles []string,
) ([]string, error) {
	if len(titles) == 0 {
		return titles, nil
	}
	out := make([]string, 0, len(titles))
	for _, title := range titles {
		if titleBlockedForMerge(ctx, repo, baseID, mergingID, comType, title) {
			continue
		}
		out = append(out, title)
	}
	return out, nil
}

func titleBlockedForMerge(
	ctx context.Context,
	repo domain.ComicRepository,
	baseID int,
	mergingID int,
	comType int,
	title string,
) bool {
	pairs := identity.DiscoveryTitleKeyPairsForTitles([]string{title}, comType)
	for _, pair := range pairs {
		matches, err := repo.FindComicsByDiscoveryTitleKeys(ctx, pair.Scope, []string{pair.TitleKey})
		if err != nil {
			return true
		}
		for _, match := range matches {
			if match.Deleted {
				continue
			}
			if match.ID == baseID || match.ID == mergingID {
				continue
			}
			return true
		}
	}
	return false
}
