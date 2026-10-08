package scrape

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchHTMLSoftFailsOnNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "missing", http.StatusNotFound)
	}))
	defer server.Close()

	fetcher := NewHTTPFetcher()
	html, err := fetcher.FetchHTML(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("expected soft fail, got error %v", err)
	}
	if html != "" {
		t.Fatalf("expected empty html, got %q", html)
	}
}

func TestFetchHTMLReturnsBodyOn200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html>ok</html>"))
	}))
	defer server.Close()

	fetcher := NewHTTPFetcher()
	html, err := fetcher.FetchHTML(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if html != "<html>ok</html>" {
		t.Fatalf("unexpected body %q", html)
	}
}

func TestSetBrowserHeadersNelomangaReferer(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://www.nelomanga.net/manga-list/latest-manga?page=2", nil)
	if err != nil {
		t.Fatal(err)
	}
	setBrowserHeaders(req, "https://www.nelomanga.net/manga-list/latest-manga?page=2")
	if got := req.Header.Get("Referer"); got != "https://www.nelomanga.net/" {
		t.Fatalf("expected nelomanga referer, got %q", got)
	}
}

func TestRunAllContinuesAfterFetchFailure(t *testing.T) {
	ctx := context.Background()
	repo := newTestScrapeRepo(t)
	fetcher := staticFetcher{
		"fixture://ok": `<div class="bsx"><a><div class="img"><img src="https://example.com/c.webp"/></div></a><div class="bigor"><div class="tt">Hero</div><div class="chapter-list"><a><div><div>Chapter 1</div></div></a></div></div></div>`,
	}
	err := RunAll(ctx, repo, fetcher)
	if err != nil {
		t.Fatalf("RunAll should not fail on empty pages: %v", err)
	}
}
