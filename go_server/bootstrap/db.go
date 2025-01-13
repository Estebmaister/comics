package bootstrap

import (
	"context"
	"fmt"
	"log"
	"time"

	"comics/mongo"
)

func NewMongoDatabase(env *Env) mongo.Client {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(env.ContextTimeout)*time.Second)
	defer cancel()

	mongodbURI := fmt.Sprintf(
		"mongodb+srv://%s:%s@%s/?retryWrites=true&w=majority&appName=Sandbox",
		env.DBUser, env.DBPass, env.DBAddr)

	if env.DBUser == "" || env.DBPass == "" {
		mongodbURI = fmt.Sprintf("mongodb://%s", env.DBAddr)
	}

	client, err := mongo.NewClient(ctx, mongodbURI)
	if err != nil {
		log.Fatalf("error connecting to mongo DB: %s", err)
	}

	err = client.Ping(ctx)
	if err != nil {
		log.Fatalf("error pinging mongo DB: %s", err)
	}

	return client
}

func CloseMongoDBConnection(client mongo.Client) {
	if client == nil {
		return
	}

	err := client.Disconnect(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Connection to MongoDB closed.")
}
