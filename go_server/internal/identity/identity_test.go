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
	if comicKey != "series:mercenarysrebirthamongnobles" {
		t.Fatalf("unexpected comic key: %q", comicKey)
	}
	if novelKey != "novel:mercenarysrebirthamongnobles" {
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

func TestTitlesMatchBySignificantTokensReorderedTitles(t *testing.T) {
	a := "Return of the frozen player"
	b := "The frozen player returns"
	if !identity.TitlesMatchBySignificantTokens(a, b) {
		t.Fatal("expected reordered frozen player titles to match")
	}
}

func TestTitlesMatchBySignificantTokensDoesNotMergeDistinctPrinceTitles(t *testing.T) {
	a := "I became the rogue first prince"
	b := "I became the first prince: legend of sword's song"
	if identity.TitlesMatchBySignificantTokens(a, b) {
		t.Fatal("expected distinct prince titles not to token-match")
	}
}

func TestBuildIdentityKeyFromTitles(t *testing.T) {
	key := identity.BuildIdentityKeyFromTitles(
		[]string{"The duke's daughter tames the beast"},
		3,
	)
	if key != "series:dukesdaughtertamesthebeast" {
		t.Fatalf("unexpected key: %q", key)
	}
}

func TestDiscoveryKeyUnifiesLeadingArticle(t *testing.T) {
	if identity.DiscoveryKey("The executioner", 3) != identity.DiscoveryKey("Executioner", 3) {
		t.Fatal("expected article-insensitive discovery keys")
	}
}
