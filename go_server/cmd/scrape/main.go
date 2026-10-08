// One-shot publisher scrape (make go-scrape).
package main

import (
	"context"
	"os"

	"comics/bootstrap"
	"comics/internal/scrape"

	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()
	app := bootstrap.MustLoadApp(ctx)
	if err := app.ScrapeCoord.Run(ctx, scrape.SourceCLI); err != nil {
		if scrape.IsScrapeInProgress(err) {
			log.Warn().Err(err).Msg("scrape skipped")
			os.Exit(2)
		}
		log.Error().Err(err).Msg("scrape failed")
		os.Exit(1)
	}
	log.Info().Msg("scrape finished")
}
