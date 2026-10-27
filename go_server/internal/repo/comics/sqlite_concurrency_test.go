package comics

import (
	"context"
	"testing"
	"time"

	"comics/domain"
)

func TestSQLiteListCompletesDuringHeldWriteTransaction(t *testing.T) {
	ctx := context.Background()
	repo := newTestSQLiteRepo(t)
	if _, err := repo.Create(ctx, domain.Comic{Titles: []string{"Hold me"}, CurrentChap: 1}); err != nil {
		t.Fatal(err)
	}

	txStarted := make(chan struct{})
	releaseTx := make(chan struct{})
	go func() {
		_ = repo.WithTx(ctx, func(txRepo domain.ComicRepository) error {
			txStarted <- struct{}{}
			<-releaseTx
			return nil
		})
	}()

	select {
	case <-txStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("write transaction did not start")
	}

	start := time.Now()
	if _, err := repo.List(ctx, domain.ComicListQuery{Limit: 12}); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	close(releaseTx)

	if elapsed > 500*time.Millisecond {
		t.Fatalf("list blocked behind writer for %s; expected concurrent read pool", elapsed)
	}
}
