package scrape

import (
	"context"
	"os"
	"testing"
)

// Run with: AUDIT_SCRAPE=1 go test ./internal/scrape/ -run AuditLivePublishers -v
func TestAuditLivePublishers(t *testing.T) {
	if os.Getenv("AUDIT_SCRAPE") != "1" {
		t.Skip("set AUDIT_SCRAPE=1 to probe live publisher pages")
	}
	ctx := context.Background()
	fetcher := NewPageFetcher()
	pairs, err := publisherURLPairs()
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range pairs {
		publisher, url := pair[0], pair[1]
		scraper := publisherScrapers[publisher]
		if scraper == nil {
			t.Logf("%s %s -> no scraper", publisher, url)
			continue
		}
		count := 0
		err := scraper(ctx, url, publisherIDs[publisher], fetcher, func(raw ScrapedComic, _ int) error {
			count++
			return nil
		})
		status := "ok"
		if err != nil {
			status = err.Error()
		}
		t.Logf("%s %s -> extracted=%d status=%s", publisher, url, count, status)
	}
}
