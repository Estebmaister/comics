// One-shot publisher scrape (legacy: make py-scrape).
package main

import (
	"context"
	"os"

	"comics/bootstrap"

	"github.com/rs/zerolog/log"
)

func main() {
	ctx := context.Background()
	app := bootstrap.MustLoadApp(ctx)
	if err := app.ComicService.Scrape(ctx); err != nil {
		log.Error().Err(err).Msg("scrape failed")
		os.Exit(1)
	}
	log.Info().Msg("scrape finished")
}
