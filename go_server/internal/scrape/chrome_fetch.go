package scrape

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

const maxFetchBodyBytes = 20 << 20

// ChromeTLSFetcher uses a Chrome TLS/JA3 profile and cookie jar, similar to Python
// cloudscraper.create_scraper(browser='chrome').
type ChromeTLSFetcher struct {
	once   sync.Once
	client tls_client.HttpClient
	initErr error
}

func NewChromeTLSFetcher() *ChromeTLSFetcher {
	return &ChromeTLSFetcher{}
}

func (f *ChromeTLSFetcher) clientOrInit() (tls_client.HttpClient, error) {
	f.once.Do(func() {
		jar := tls_client.NewCookieJar()
		f.client, f.initErr = tls_client.NewHttpClient(
			tls_client.NewNoopLogger(),
			tls_client.WithTimeoutSeconds(int(requestTimeout.Seconds())),
			tls_client.WithClientProfile(profiles.Chrome_120),
			tls_client.WithCookieJar(jar),
		)
	})
	return f.client, f.initErr
}

func nelomangaSiteURL(pageURL string) bool {
	return strings.Contains(strings.ToLower(pageURL), "nelomanga.net")
}

func nelomangaListingURL(pageURL string) bool {
	lower := strings.ToLower(pageURL)
	return strings.Contains(lower, "nelomanga.net") &&
		(strings.Contains(lower, "/manga-list/") || strings.Contains(lower, "/genre/"))
}

// NewPageFetcher is the default scrape transport (Chrome TLS client).
func NewPageFetcher() PageFetcher {
	return NewChromeTLSFetcher()
}

func (f *ChromeTLSFetcher) FetchHTML(ctx context.Context, url string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if nelomangaListingURL(url) {
		if _, err := f.fetchURL(ctx, "https://www.nelomanga.net/"); err != nil {
			return "", err
		}
	}
	return f.fetchURL(ctx, url)
}

func (f *ChromeTLSFetcher) fetchURL(ctx context.Context, url string) (string, error) {
	client, err := f.clientOrInit()
	if err != nil {
		slog.Warn("scrape tls client init failed", "url", url, "error", err)
		return "", nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	setFHTTPBrowserHeaders(req, url)
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", err
		}
		slog.Warn("scrape fetch failed", "url", url, "error", err)
		return "", nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		slog.Warn("scrape fetch non-200", "url", url, "status", resp.StatusCode)
		return "", nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchBodyBytes))
	if err != nil {
		if ctx.Err() != nil {
			return "", err
		}
		slog.Warn("scrape fetch read failed", "url", url, "error", err)
		return "", nil
	}
	html := string(body)
	if strings.Contains(html, "Just a moment") {
		slog.Warn("scrape fetch cloudflare challenge", "url", url)
		return "", nil
	}
	return html, nil
}

func setFHTTPBrowserHeaders(req *http.Request, pageURL string) {
	req.Header.Set("User-Agent", defaultChromeUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	if nelomangaSiteURL(pageURL) {
		req.Header.Set("Referer", "https://www.nelomanga.net/")
	}
}
