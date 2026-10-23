package comics

import (
	"context"
	"testing"

	"comics/domain"
	"comics/internal/identity"
)

func TestSQLComicRepositoryStoresDiscoveryKeys(t *testing.T) {
	repo := newTestSQLiteRepo(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, domain.Comic{
		Titles:      []string{"The executioner", "Executioner"},
		CurrentChap: 80,
		ComType:     3,
	})
	if err != nil {
		t.Fatal(err)
	}

	found, err := repo.FindComicsByDiscoveryTitleKeys(ctx, "series", []string{
		identity.DiscoveryTitleKey("Executioner"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].ID != created.ID {
		t.Fatalf("unexpected lookup result %#v", found)
	}
}
