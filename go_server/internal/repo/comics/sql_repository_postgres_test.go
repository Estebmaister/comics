package comics

import (
	"context"
	"os"
	"testing"

	"comics/domain"
)

func TestPostgresComicRepositoryIdentityKey(t *testing.T) {
	url := os.Getenv("COMICS_POSTGRES_TEST_URL")
	if url == "" {
		t.Skip("COMICS_POSTGRES_TEST_URL not set")
	}
	ctx := context.Background()
	repo, err := NewPostgresComicRepository(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	created, err := repo.Create(ctx, domain.Comic{
		Titles:      []string{"Postgres hero"},
		CurrentChap: 5,
		ComType:     3,
	})
	if err != nil {
		t.Fatal(err)
	}
	key := comicIdentityKey(created)
	found, err := repo.GetByIdentityKey(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if found.ID != created.ID {
		t.Fatalf("expected identity lookup for %q, got %#v", key, found)
	}
}
