package scrape

import (
	"context"
	"log/slog"
)

func registerWithLogging(
	ctx context.Context,
	publisherName string,
	registrar *Registrar,
) func(ScrapedComic, int) error {
	return func(raw ScrapedComic, publisherID int) error {
		err := registrar.Register(ctx, raw, publisherID)
		if err != nil && ctx.Err() == nil {
			slog.Warn("scrape register failed",
				"publisher", publisherName,
				"title", raw.Title,
				"error", err,
			)
		}
		return err
	}
}
