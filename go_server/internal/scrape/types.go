package scrape

import "context"

type ScrapedComic struct {
	Chapter  string
	Title    string
	CoverURL string
	ComType  string
	Status   string
	Author   string
}

type PageFetcher interface {
	FetchHTML(ctx context.Context, url string) (string, error)
}

type PublisherScraper func(
	ctx context.Context,
	url string,
	publisherID int,
	fetcher PageFetcher,
	register func(ScrapedComic, int) error,
) error
