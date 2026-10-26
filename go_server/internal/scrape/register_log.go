package scrape

import (
	"context"
	"errors"
	"log/slog"

	"comics/domain"
)

func registerWithLogging(
	ctx context.Context,
	publisherName string,
	registrar *Registrar,
) func(ScrapedComic, int) error {
	return func(raw ScrapedComic, publisherID int) error {
		err := registrar.Register(ctx, raw, publisherID)
		if err != nil && ctx.Err() == nil {
			if errors.Is(err, domain.ErrAmbiguousComicMatch) {
				var ambiguous *domain.AmbiguousComicMatchError
				if errors.As(err, &ambiguous) {
					slog.Info("scrape register ambiguous match",
						"publisher", publisherName,
						"title", raw.Title,
						"scope", ambiguous.Scope,
						"match_kind", ambiguous.MatchKind,
						"candidate_ids", ambiguous.CandidateIDs,
					)
				}
				return err
			}
			slog.Warn("scrape register failed",
				"publisher", publisherName,
				"title", raw.Title,
				"error", err,
			)
		}
		return err
	}
}
