package bootstrap

import (
	"context"
	"log"
	"time"

	"comics/internal/repo/mongo"
)

func NewRepo(env *Env) (ClosableUserStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(env.ContextTimeout)*time.Second)
	defer cancel()

	cfg := &env.DB
	return mongo.NewUserRepo(ctx, cfg)
}

// CloseConnection safely disconnects from the repo DB
func CloseConnection(ctx context.Context, repo Closable, duration time.Duration) error {
	if err := repo.Close(ctx, duration); err != nil {
		log.Printf("Error closing DB connection: %v", err)
		return err
	}

	log.Println("Connection to DB closed successfully.")
	return nil
}
