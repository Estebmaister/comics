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

// Publishers whose listing pages share Cloudflare cookies across paginated URLs.
var sequentialURLPublishers = map[string]bool{
	"Manganato": true,
}

// RunAll executes every active publisher scraper against the comics repository.
// Per-URL failures are logged and skipped, matching Python asyncio.gather behavior.
func RunAll(ctx context.Context, repo domain.ComicRepository, fetcher PageFetcher) error {
	if fetcher == nil {
		fetcher = NewPageFetcher()
	}
	pairs, err := publisherURLPairs()
	if err != nil {
		return err
	}
	registrar := NewRegistrar(repo, NewRunState())
	var registerMu sync.Mutex

	byPublisher := map[string][]string{}
	for _, pair := range pairs {
		byPublisher[pair[0]] = append(byPublisher[pair[0]], pair[1])
	}

	var wg sync.WaitGroup
	for publisherName, urls := range byPublisher {
		scraper := publisherScrapers[publisherName]
		publisherID := publisherIDs[publisherName]
		if scraper == nil {
			continue
		}
		if sequentialURLPublishers[publisherName] {
			wg.Add(1)
			go func(publisherName string, urls []string, scraper PublisherScraper, publisherID int) {
				defer wg.Done()
				registerFn := makeRegisterFn(ctx, publisherName, registrar, &registerMu)
				for _, url := range urls {
					if err := scraper(ctx, url, publisherID, fetcher, registerFn); err != nil {
						if ctx.Err() != nil {
							return
						}
						slog.Warn("scrape publisher task failed",
							"publisher", publisherName,
							"url", url,
							"error", err,
						)
					}
				}
			}(publisherName, urls, scraper, publisherID)
			continue
		}
		for _, url := range urls {
			wg.Add(1)
			go func(publisherName, url string, scraper PublisherScraper, publisherID int) {
				defer wg.Done()
				registerFn := makeRegisterFn(ctx, publisherName, registrar, &registerMu)
				if err := scraper(ctx, url, publisherID, fetcher, registerFn); err != nil {
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
	}
	wg.Wait()
	return nil
}

func makeRegisterFn(
	ctx context.Context,
	publisherName string,
	registrar *Registrar,
	registerMu *sync.Mutex,
) func(ScrapedComic, int) error {
	register := registerWithLogging(ctx, publisherName, registrar)
	return func(raw ScrapedComic, pubID int) error {
		registerMu.Lock()
		defer registerMu.Unlock()
		return register(raw, pubID)
	}
}
