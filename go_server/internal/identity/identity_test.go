package identity_test

import (
	"testing"

	"comics/internal/identity"
)

func TestNovelAndSeriesIdentityKeysDiffer(t *testing.T) {
	title := "A mercenary's rebirth among nobles"
	comicKey := identity.BuildIdentityKey(title, 3)
	novelKey := identity.BuildIdentityKey(title, identity.NovelType)

	if comicKey == novelKey {
		t.Fatalf("expected different keys for comic vs novel")
	}
	if comicKey != "series:a mercenary's rebirth among nobles" {
		t.Fatalf("unexpected comic key: %q", comicKey)
	}
	if novelKey != "novel:a mercenary's rebirth among nobles" {
		t.Fatalf("unexpected novel key: %q", novelKey)
	}
}

func TestNormalizeTitleVariantsUsesSentenceCase(t *testing.T) {
	titles := identity.NormalizeTitleVariants(
		[]string{"THE WORLD'S BEST ENGINEER", "the world's best engineer"},
		3,
	)
	if len(titles) != 1 || titles[0] != "The world's best engineer" {
		t.Fatalf("unexpected titles: %#v", titles)
	}
}

func TestTitlesArePrefixMatchTruncatedScrape(t *testing.T) {
	short := "The strongest assassin gets transferr"
	long := "The strongest assassin gets transferred to another world with his whole class"
	if !identity.TitlesArePrefixMatch(short, long) {
		t.Fatal("expected truncated title to prefix-match full title")
	}
}

func TestBuildIdentityKeyFromTitles(t *testing.T) {
	key := identity.BuildIdentityKeyFromTitles(
		[]string{"The duke's daughter tames the beast"},
		3,
	)
	if key != "series:the duke's daughter tames the beast" {
		t.Fatalf("unexpected key: %q", key)
	}
}
