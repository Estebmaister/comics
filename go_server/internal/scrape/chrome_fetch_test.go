package scrape

import "testing"

func TestNelomangaURLHelpers(t *testing.T) {
	if !nelomangaSiteURL("https://www.nelomanga.net/") {
		t.Fatal("expected nelomanga site")
	}
	if !nelomangaListingURL("https://www.nelomanga.net/manga-list/latest-manga?page=1") {
		t.Fatal("expected listing url")
	}
	if nelomangaListingURL("https://www.nelomanga.net/") {
		t.Fatal("home should not be listing")
	}
}

func TestNewPageFetcherIsChromeTLS(t *testing.T) {
	if _, ok := NewPageFetcher().(*ChromeTLSFetcher); !ok {
		t.Fatal("expected ChromeTLSFetcher as default page fetcher")
	}
}
