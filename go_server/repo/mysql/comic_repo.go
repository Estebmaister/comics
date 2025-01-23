package mysql

import (
	"context"
	"database/sql"

	"comics/sql/mysql"
)

type Repo struct {
	db      *sql.DB
	queries *mysql.Queries
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{
		db:      db,
		queries: mysql.New(db),
	}
}

func (r *Repo) WithTx(ctx context.Context, fn func(*mysql.Queries) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create a new Queries instance with the transaction
	qtx := r.queries.WithTx(tx)

	// Execute the function with the transactional queries
	if err := fn(qtx); err != nil {
		return err
	}

	// Commit the transaction
	return tx.Commit()
}
