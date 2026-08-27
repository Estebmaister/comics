package comics

import (
	"context"
	"testing"

	"comics/domain"
	"comics/internal/identity"
)

func TestSQLComicRepositoryStoresIdentityKey(t *testing.T) {
	ctx := context.Background()
	repo := newTestSQLiteRepo(t)
	created, err := repo.Create(ctx, domain.Comic{
		Titles:      []string{"Identity hero"},
		CurrentChap: 3,
		ComType:     3,
	})
	if err != nil {
		t.Fatal(err)
	}
	expected := identity.BuildIdentityKeyFromTitles(created.Titles, created.ComType)
	found, err := repo.GetByIdentityKey(ctx, expected)
	if err != nil {
		t.Fatal(err)
	}
	if found.ID != created.ID {
		t.Fatalf("expected comic %d, got %d", created.ID, found.ID)
	}
}
