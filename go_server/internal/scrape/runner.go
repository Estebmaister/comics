package scrape

import (
	"context"
	"log/slog"
	"sync"

	"comics/domain"
)

var publisherScrapers = map[string]PublisherScraper{
	"Asura":        scrapeAsuraPage,
	"ManhuaPlus":   scrapeManhuaPlusPage,
	"FlameScans":   scrapeFlamePage,
	"RealmScans":   scrapeRealmPage,
	"DemonicScans": scrapeDemonicPage,
	"Manganato":    scrapeManganatoPage,
}

var publisherIDs = map[string]int{
	"Asura":        PublisherAsura,
	"ManhuaPlus":   PublisherManhuaPlus,
	"FlameScans":   PublisherFlameScans,
	"RealmScans":   PublisherRealmScans,
	"Manganato":    PublisherManganato,
	"DemonicScans": PublisherDemonicScans,
}

// RunAll executes every active publisher scraper against the comics repository.
// Per-URL failures are logged and skipped, matching Python asyncio.gather behavior.
func RunAll(ctx context.Context, repo domain.ComicRepository, fetcher PageFetcher) error {
	if fetcher == nil {
		fetcher = NewHTTPFetcher()
	}
	pairs, err := publisherURLPairs()
	if err != nil {
		return err
	}
	registrar := NewRegistrar(repo, NewRunState())
	var wg sync.WaitGroup
	for _, pair := range pairs {
		publisherName, url := pair[0], pair[1]
		scraper := publisherScrapers[publisherName]
		publisherID := publisherIDs[publisherName]
		if scraper == nil {
			continue
		}
		wg.Add(1)
		go func(publisherName, url string, scraper PublisherScraper, publisherID int) {
			defer wg.Done()
			if err := scraper(ctx, url, publisherID, fetcher, func(raw ScrapedComic, pubID int) error {
				return registrar.Register(ctx, raw, pubID)
			}); err != nil {
				if ctx.Err() != nil {
					return
				}
				slog.Warn("scrape publisher task failed",
					"publisher", publisherName,
					"url", url,
					"error", err,
				)
			}
		}(publisherName, url, scraper, publisherID)
	}
	wg.Wait()
	return nil
}
