package comics

import (
	"os"
	"testing"
)

func TestPostgresDiscoveryContract(t *testing.T) {
	url := os.Getenv("COMICS_POSTGRES_URL")
	if url == "" {
		t.Skip("COMICS_POSTGRES_URL not set")
	}
	repo, err := NewPostgresComicRepository(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	runDiscoveryContract(t, repo)
}
