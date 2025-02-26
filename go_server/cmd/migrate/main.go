package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

type Direction string

const (
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"
)

// main configure only for PostgreSQL
func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	if len(os.Args) < 2 {
		log.Fatal("Wrong Usage, command should be: migrate [up|down]")
	}

	err := Migrate(context.Background(), "internal/db/migrations",
		&ConfigDB{
			Addr: os.Getenv("PG_ADDR"),
			Host: os.Getenv("PG_HOST"),
			Port: os.Getenv("PG_PORT"),
			User: os.Getenv("PG_USER"),
			Pass: os.Getenv("PG_PASS"),
			Name: os.Getenv("PG_NAME")},
		Direction(os.Args[1]), WithLogger(stdLogger{}))
	if err != nil {
		log.Fatal(err)
	}
}

type Opt func(*config) error

func WithLogger(l logger) Opt {
	return func(cfg *config) error {
		cfg.Logger = l
		return nil
	}
}

type logger interface {
	Log(ctx context.Context, fmt string, args ...any)
}

type stdLogger struct{}

func (s stdLogger) Log(_ context.Context, fmt string, args ...any) {
	log.Printf(fmt, args...)
}

type config struct {
	MigrationsDirectory string
	DBConfig            *ConfigDB
	Logger              logger
}

// ConfigDB defines options for Database
type ConfigDB struct {
	Name string `toml:"db_name" mapstructure:"db_name"`
	User string `toml:"db_user" mapstructure:"db_user"`
	Pass string `toml:"db_password" mapstructure:"db_password"`
	Host string `toml:"db_host" mapstructure:"db_host"`
	Port string `toml:"db_port" mapstructure:"db_port"`

	// Addr is placed in the host section of the DSN, and it can hold either
	// the host and port ("localhost:5432") or the unix socket address
	// ("/db.sock"). For sqlite3 databases, this is a file name or `:memory:`.
	Addr string `toml:"addr" mapstructure:"addr"`
}

// Migrate configure only for PostgreSQL
func Migrate(
	ctx context.Context, migrations string, dbConfig *ConfigDB, direction Direction, opts ...Opt) error {
	cfg := &config{
		MigrationsDirectory: migrations,
		DBConfig:            dbConfig,
		Logger:              stdLogger{},
	}

	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return fmt.Errorf("migration failed: failed to apply option: %w", err)
		}
	}

	m, err := migrate.New(
		fmt.Sprintf("file://%s", cfg.MigrationsDirectory),
		getDBConnURL(cfg.DBConfig))
	if err != nil {
		cfg.Logger.Log(ctx, "failed to prepare the migration: %v", err)
		return fmt.Errorf("migration failed: %w", err)
	}

	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			m.GracefulStop <- true
		}
	}()

	var version int
	switch direction {
	case DirectionUp:
		err = m.Up()
	case DirectionDown:
		err = m.Down()
	case "force": // implement
		err = m.Force(version)
	case "step": // implement
		err = m.Steps(version)
	default:
		err = fmt.Errorf("invalid direction. Use 'up' or 'down'")
	}
	if err != nil {
		switch err {
		case migrate.ErrNoChange:
		default:
			cfg.Logger.Log(ctx, "failed to execute the migration: %v", err)
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	cfg.Logger.Log(ctx, "migration finished")
	return nil
}

func getDBConnURL(cfg *ConfigDB) string {
	user := url.QueryEscape(cfg.User)
	pwd := url.QueryEscape(cfg.Pass)
	addr := cfg.Addr
	if addr == "" {
		addr = fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=require",
		user, pwd, addr, cfg.Name)
}
