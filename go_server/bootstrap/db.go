package bootstrap

import (
	"context"
	"fmt"
	"log"
	"time"

	"comics/domain"
	"comics/internal/repo/mongo"
)

func NewMongoRepo(env *Env) domain.UserStore {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(env.ContextTimeout)*time.Second)
	defer cancel()

	uri := fmt.Sprintf(
		"mongodb+srv://%s:%s@%s/?retryWrites=true&w=majority&appName=Sandbox",
		env.DBUser, env.DBPass, env.DBAddr)

	if env.DBUser == "" || env.DBPass == "" {
		uri = fmt.Sprintf("mongodb://%s", env.DBAddr)
	}

	var cfg *mongo.DatabaseConfig
	client, err := mongo.NewMongoClient(ctx, cfg, uri)
	if err != nil {
		log.Fatalf("error connecting to mongo DB: %s", err)
	}

	err = client.Ping(ctx)
	if err != nil {
		log.Fatalf("error pinging mongo DB: %s", err)
	}

	return mongo.NewUserRepo(client, env.DBName, env.DBCollection)
}

func CloseMongoDBConnection(repo domain.UserStore) {
	if repo == nil {
		return
	}

	err := repo.(*mongo.UserRepo).Client().Disconnect(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Connection to MongoDB closed.")
}
