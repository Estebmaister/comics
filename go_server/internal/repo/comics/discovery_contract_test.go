package comics

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"comics/domain"
	"comics/internal/identity"
)

func runDiscoveryContract(t *testing.T, repo *SQLComicRepository) {
	t.Helper()
	ctx := context.Background()
	prefix := fmt.Sprintf("contract-%d-", time.Now().UnixNano())

	t.Run("normalizes and stores every active key", func(t *testing.T) {
		created, err := repo.Create(ctx, domain.Comic{
			Titles:  []string{prefix + "ALPHA", prefix + "alpha alias"},
			ComType: 3,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = repo.Delete(ctx, created.ID) })
		if created.Titles[0] != identity.NormalizeTitleForStorage(prefix+"ALPHA", 3, true) {
			t.Fatalf("title was not normalized: %#v", created.Titles)
		}
		for _, pair := range identity.DiscoveryTitleKeyPairsForTitles(created.Titles, created.ComType) {
			found, err := repo.FindComicsByDiscoveryTitleKeys(ctx, pair.Scope, []string{pair.TitleKey})
			if err != nil || len(found) != 1 || found[0].ID != created.ID {
				t.Fatalf("missing key %#v: %#v err=%v", pair, found, err)
			}
		}
	})

	t.Run("conflict is typed and row plus old keys roll back", func(t *testing.T) {
		owner, err := repo.Create(ctx, domain.Comic{Titles: []string{prefix + "owner"}, ComType: 3})
		if err != nil {
			t.Fatal(err)
		}
		candidate, err := repo.Create(ctx, domain.Comic{Titles: []string{prefix + "candidate"}, ComType: 3})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = repo.Delete(ctx, owner.ID)
			_ = repo.Delete(ctx, candidate.ID)
		})
		candidate.Titles = owner.Titles
		_, err = repo.Update(ctx, candidate)
		var conflict *domain.DiscoveryKeyConflictError
		if !errors.As(err, &conflict) || conflict.OwnerComicID != owner.ID {
			t.Fatalf("expected owner %d conflict, got %#v / %v", owner.ID, conflict, err)
		}
		stored, err := repo.Get(ctx, candidate.ID)
		if err != nil || stored.Titles[0] != identity.NormalizeTitleForStorage(prefix+"candidate", 3, true) {
			t.Fatalf("rollback mismatch: %#v err=%v", stored, err)
		}
	})

	t.Run("self update is not a duplicate", func(t *testing.T) {
		comic, err := repo.Create(ctx, domain.Comic{Titles: []string{prefix + "self"}, ComType: 3})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = repo.Delete(ctx, comic.ID) })
		comic.Titles = []string{strings.ToUpper(prefix + "self")}
		if _, err := repo.Update(ctx, comic); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("soft and hard delete release keys", func(t *testing.T) {
		comic, err := repo.Create(ctx, domain.Comic{Titles: []string{prefix + "released"}, ComType: 3})
		if err != nil {
			t.Fatal(err)
		}
		comic.Deleted = true
		if _, err := repo.Update(ctx, comic); err != nil {
			t.Fatal(err)
		}
		key := identity.DiscoveryTitleKey(prefix + "released")
		found, err := repo.FindComicsByDiscoveryTitleKeys(ctx, "series", []string{key})
		if err != nil || len(found) != 0 {
			t.Fatalf("soft-deleted key remains: %#v err=%v", found, err)
		}
		if err := repo.Delete(ctx, comic.ID); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("unicode-only title receives a key", func(t *testing.T) {
		comic, err := repo.Create(ctx, domain.Comic{Titles: []string{"菜鳥白魔法師" + prefix}, ComType: 3})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = repo.Delete(ctx, comic.ID) })
		if identity.DiscoveryTitleKey(comic.Titles[0]) == "" {
			t.Fatal("unicode title produced an empty key")
		}
	})
}

func TestSQLiteDiscoveryContract(t *testing.T) {
	runDiscoveryContract(t, newTestSQLiteRepo(t))
}
