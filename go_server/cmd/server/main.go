package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"comics/api/route"
	"comics/bootstrap"
	_ "comics/docs"
	repo "comics/repository"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

type Application struct {
	Env   *Env
	Mongo mongo.Client
}

type Env struct {
	AppEnv         string `mapstructure:"APP_ENV"`
	ServerAddress  string `mapstructure:"SERVER_ADDRESS"`
	ContextTimeout int    `mapstructure:"CONTEXT_TIMEOUT"`
	DBHost         string `mapstructure:"DB_HOST"`
	DBPort         string `mapstructure:"DB_PORT"`
}

//	@title			Comics API
//	@version		1.1
//	@description	Server documentation to query comics from the DB.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	Estebmaister
//	@contact.url	http://www.github.com/estebmaister
//	@contact.email	estebmaister@gmail.com

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@securityDefinitions.apikey	Bearer JWT
//	@in							header
//	@name						Authorization

// @host		localhost:8081
// @BasePath	/
func main() {
	// app is the instance of the entire application, managing key resources throughout its lifecycle
	app := bootstrap.App()

	// Configuration variables
	env := app.Env

	// Database instance
	db := app.Mongo.Database(env.DBName)
	defer app.CloseDBConnection()

	timeout := time.Duration(env.ContextTimeout) * time.Second

	// Creating a gin instance
	gin := gin.Default()

	// Route binding
	route.Setup(env, timeout, db, gin)

	// Running the server
	go func() {
		gin.Run(env.ServerAddress)
	}()

	// Initialize the database
	_, err := repo.NewSQLiteDB("comics.db")
	if err != nil {
		panic("Failed to initialize SQLite database")
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}
