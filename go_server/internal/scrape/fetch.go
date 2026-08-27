package scrape

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	requestTimeout       = 10 * time.Second
	defaultChromeUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

type HTTPFetcher struct {
	Client *http.Client
}

func NewHTTPFetcher() *HTTPFetcher {
	return &HTTPFetcher{
		Client: &http.Client{Timeout: requestTimeout},
	}
}

func setBrowserHeaders(req *http.Request, pageURL string) {
	req.Header.Set("User-Agent", defaultChromeUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	if strings.Contains(pageURL, "nelomanga.net") {
		req.Header.Set("Referer", "https://www.nelomanga.net/")
	}
}

// FetchHTML loads a publisher page. Matches Python scrape_url behavior: network and
// non-200 responses are logged and yield empty HTML instead of failing the scrape run.
func (f *HTTPFetcher) FetchHTML(ctx context.Context, url string) (string, error) {
	client := f.Client
	if client == nil {
		client = &http.Client{Timeout: requestTimeout}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	setBrowserHeaders(req, url)
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if ctx.Err() != nil {
			return "", err
		}
		slog.Warn("scrape fetch read failed", "url", url, "error", err)
		return "", nil
	}
	return string(body), nil
}
