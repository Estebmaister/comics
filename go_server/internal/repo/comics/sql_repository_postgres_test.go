package comics

import (
	"context"
	"os"
	"testing"

	"comics/domain"
	"comics/internal/identity"
)

func TestPostgresComicRepositoryDiscoveryKeys(t *testing.T) {
	url := os.Getenv("COMICS_POSTGRES_URL")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("COMICS_POSTGRES_URL not set")
	}

	repo, err := NewPostgresComicRepository(url)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()

	ctx := context.Background()
	created, err := repo.Create(ctx, domain.Comic{
		Titles:      []string{"Postgres discovery hero"},
		CurrentChap: 1,
		ComType:     3,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = repo.Delete(ctx, created.ID) }()

	found, err := repo.FindComicsByDiscoveryTitleKeys(ctx, "series", []string{
		identity.DiscoveryTitleKey(created.Titles[0]),
	})
	if err != nil || len(found) != 1 || found[0].ID != created.ID {
		t.Fatalf("lookup failed: %#v err=%v", found, err)
	}
}
