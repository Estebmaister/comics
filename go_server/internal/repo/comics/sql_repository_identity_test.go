package comics

import (
	"context"
	"errors"
	"testing"

	"comics/domain"
	"comics/internal/identity"

	"github.com/google/go-cmp/cmp"
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

func TestSQLiteDiscoveryOwnershipIsTypedAndAtomic(t *testing.T) {
	ctx := context.Background()
	repo := newTestSQLiteRepo(t)

	owner, err := repo.Create(ctx, domain.Comic{
		Titles:  []string{"Alpha owner"},
		ComType: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := repo.Create(ctx, domain.Comic{
		Titles:  []string{"Beta candidate"},
		ComType: 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	sameOwner := owner
	sameOwner.Titles = []string{"THE ALPHA OWNER"}
	if _, err := repo.Update(ctx, sameOwner); err != nil {
		t.Fatalf("self-owned key should update: %v", err)
	}

	conflicting := candidate
	conflicting.Titles = []string{"The alpha owner"}
	_, err = repo.Update(ctx, conflicting)
	var conflict *domain.DiscoveryKeyConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected typed conflict, got %v", err)
	}
	if conflict.ComicID != candidate.ID || conflict.OwnerComicID != owner.ID ||
		conflict.Scope != "series" || conflict.TitleKey != "alphaowner" {
		t.Fatalf("unexpected conflict: %#v", conflict)
	}

	stored, err := repo.Get(ctx, candidate.ID)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff([]string{"Beta candidate"}, stored.Titles); diff != "" {
		t.Fatalf("comic row changed despite conflict (-want +got):\n%s", diff)
	}
	beta, err := repo.FindComicsByDiscoveryTitleKeys(ctx, "series", []string{"betacandidate"})
	if err != nil || len(beta) != 1 || beta[0].ID != candidate.ID {
		t.Fatalf("candidate key did not roll back: %#v err=%v", beta, err)
	}
}

func TestSQLiteSoftDeleteReleasesAndUndeleteReacquiresKeys(t *testing.T) {
	ctx := context.Background()
	repo := newTestSQLiteRepo(t)
	owner, err := repo.Create(ctx, domain.Comic{Titles: []string{"Released key"}, ComType: 3})
	if err != nil {
		t.Fatal(err)
	}
	owner.Deleted = true
	if _, err := repo.Update(ctx, owner); err != nil {
		t.Fatal(err)
	}
	found, err := repo.FindComicsByDiscoveryTitleKeys(ctx, "series", []string{"releasedkey"})
	if err != nil || len(found) != 0 {
		t.Fatalf("deleted owner retained a key: %#v err=%v", found, err)
	}

	replacement, err := repo.Create(ctx, domain.Comic{Titles: []string{"Released key"}, ComType: 3})
	if err != nil {
		t.Fatal(err)
	}
	owner.Deleted = false
	if _, err := repo.Update(ctx, owner); !errors.Is(err, domain.ErrDiscoveryKeyConflict) {
		t.Fatalf("undelete should conflict with comic %d: %v", replacement.ID, err)
	}
	stored, err := repo.Get(ctx, owner.ID)
	if err != nil || !stored.Deleted {
		t.Fatalf("failed undelete should roll back: %#v err=%v", stored, err)
	}
}
