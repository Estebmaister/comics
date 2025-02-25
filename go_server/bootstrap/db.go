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

	return mongo.NewUserRepo(ctx, &env.DB)
}

// CloseConnection safely disconnects from the repo DB
func CloseConnection(ctx context.Context, repo Closable) error {
	if err := repo.Close(ctx); err != nil {
		log.Printf("Error closing DB connection: %v", err)
		return err
	}

	log.Println("Connection to DB closed successfully.")
	return nil
}
