package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"comics/internal/logger"
	"comics/internal/repo"
	"comics/internal/tracer"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// AppEnv represents the application's runtime environment
type AppEnv string

// AppEnv values
const (
	Development AppEnv = "development"
	Production  AppEnv = "production"
	Testing     AppEnv = "testing"

	defaultEnv        = Production
	defaultCtxTimeout = 10 * time.Second
	defaultHTTPAddr   = "localhost"
	defaultHTTPPort   = "8081"
	defaultGRPCAddr   = "localhost"
	defaultGRPCPort   = "8082"
	defaultTLSCert    = "../tls/comics.crt"
	defaultTLSKey     = "../tls/comics.key"
)

// Env holds the application configuration
type Env struct {
	DBConfig           *repo.DBConfig
	JWTConfig          *JWTConfig
	LoggerConfig       *logger.Config
	TracerConfig       *tracer.Config
	AppEnv             `mapstructure:"ENVIRONMENT"`
	GoogleClientID     string        `mapstructure:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string        `mapstructure:"GOOGLE_CLIENT_SECRET"`
	AddressHTTP        string        `mapstructure:"HTTP_ADDRESS"`
	PortHTTP           string        `mapstructure:"HTTP_PORT"`
	AddressGRPC        string        `mapstructure:"GRPC_ADDRESS"`
	PortGRPC           string        `mapstructure:"GRPC_PORT"`
	HostURL            string        `mapstructure:"HOST_URL"`
	ComicsDBDriver     string        `mapstructure:"COMICS_DB_DRIVER"`
	ComicsSQLitePath   string        `mapstructure:"COMICS_SQLITE_PATH"`
	ComicsPostgresURL  string        `mapstructure:"COMICS_POSTGRES_URL"`
	PythonBackendURL   string        `mapstructure:"PY_BACKEND_URL"`
	CORSAllowedOrigins string        `mapstructure:"CORS_ALLOWED_ORIGINS"`
	HTTPTLSCertFile    string        `mapstructure:"HTTP_TLS_CERT_FILE"`
	HTTPTLSKeyFile     string        `mapstructure:"HTTP_TLS_KEY_FILE"`
	InitCtxTimeout     time.Duration `mapstructure:"INIT_TIMEOUT"`
	// SCRAPE_INTERVAL enables a background scrape loop in cmd/server (0 = off).
	// Python parity: standalone scrape uses 10m; py-daemon server+scrape uses 100m.
	ScrapeInterval time.Duration `mapstructure:"SCRAPE_INTERVAL"`
}

// JWTConfig holds the configuration for the JW Token
type JWTConfig struct {
	AccessTokenExpiryHour  time.Duration `mapstructure:"ACCESS_TOKEN_EXPIRY_HOUR"`
	RefreshTokenExpiryHour time.Duration `mapstructure:"REFRESH_TOKEN_EXPIRY_HOUR"`
	AccessTokenSecret      string        `mapstructure:"ACCESS_TOKEN_SECRET"`
	RefreshTokenSecret     string        `mapstructure:"REFRESH_TOKEN_SECRET"`
}

// MustLoadEnv reads the environment configuration with viper
func MustLoadEnv(_ context.Context) *Env {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env") // Define the config type as ENV
	viper.AutomaticEnv()       // Priority to read from environment variables
	setDefaults()
	bindEnvKeys()
	env := Env{}
	jwtConfig := &JWTConfig{}
	dbConfig := &repo.DBConfig{}
	loggerConfig := &logger.Config{}
	tracerConfig := &tracer.Config{}

	err := viper.ReadInConfig()
	if err != nil {
		dir, errDir := os.Getwd()
		log.Warn().Err(err).Msgf("Can't find the file .env in %s, using environment variables and defaults", dir)
		if errDir != nil {
			log.Warn().Err(errDir).Msg("Can't get the current directory")
		}
	}

	errs := []error{}
	errs = append(errs, viper.Unmarshal(&env))
	errs = append(errs, viper.Unmarshal(&jwtConfig))
	errs = append(errs, viper.Unmarshal(&dbConfig))
	errs = append(errs, viper.Unmarshal(&loggerConfig))
	errs = append(errs, viper.Unmarshal(&tracerConfig))
	for _, err := range errs {
		if err != nil {
			log.Fatal().Err(err).Msg("Environment can't be Unmarshal from Viper")
		}
	}
	env.DBConfig = dbConfig
	env.JWTConfig = jwtConfig
	env.LoggerConfig = loggerConfig
	env.TracerConfig = tracerConfig

	// Cast the application environment to a type
	env.AppEnv = parseAppEnv(string(env.AppEnv))
	env.applyLocalTLSDefaults()
	if !env.AppEnv.IsProduction() {
		log.Debug().Msg("The App is running in a dev env")
		log.Debug().Msgf("%v\n", Sanitize(env))
	}

	// Set the default initialization timeout
	if env.InitCtxTimeout == 0 {
		env.InitCtxTimeout = defaultCtxTimeout
	}
	if err := env.Validate(); err != nil {
		log.Fatal().Err(err).Msg("Invalid environment configuration")
	}

	return &env
}

func setDefaults() {
	viper.SetDefault("ENVIRONMENT", string(Development))
	viper.SetDefault("HTTP_ADDRESS", defaultHTTPAddr)
	viper.SetDefault("HTTP_PORT", defaultHTTPPort)
	viper.SetDefault("GRPC_ADDRESS", defaultGRPCAddr)
	viper.SetDefault("GRPC_PORT", defaultGRPCPort)
	viper.SetDefault("HOST_URL", defaultHTTPAddr+":"+defaultHTTPPort)
	viper.SetDefault("INIT_TIMEOUT", defaultCtxTimeout)
	viper.SetDefault("SCRAPE_INTERVAL", time.Duration(0))
	viper.SetDefault("COMICS_DB_DRIVER", "sqlite")
	viper.SetDefault("COMICS_SQLITE_PATH", filepath.Join("..", "src", "db", "comics.db"))
	viper.SetDefault("CORS_ALLOWED_ORIGINS",
		"http://localhost:3000,https://localhost:3000,http://127.0.0.1:3000,https://127.0.0.1:3000,https://estebmaister.github.io")
	viper.SetDefault("DB_ADDR", "localhost:27017")
	viper.SetDefault("DB_NAME", "comics")
	viper.SetDefault("DB_TABLE_USERS", "users")
	viper.SetDefault("DB_TABLE_COMICS", "comics")
	viper.SetDefault("DB_MAX_POOL_SIZE", 100)
	viper.SetDefault("DB_MIN_POOL_SIZE", 0)
	viper.SetDefault("DB_MAX_CONN_IDLE_TIME", 5*time.Minute)
	viper.SetDefault("DB_CONN_TIMEOUT", 30*time.Second)
	viper.SetDefault("DB_BACKOFF_TIMEOUT", 15*time.Second)
	viper.SetDefault("ACCESS_TOKEN_EXPIRY_HOUR", time.Hour)
	viper.SetDefault("REFRESH_TOKEN_EXPIRY_HOUR", 168*time.Hour)
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_DEBUG_CONSOLE", true)
	viper.SetDefault("LOG_OUTPUT_FILE", ".observability/log/logs.log")
	viper.SetDefault("LOG_MAX_SIZE_MB", 1)
	viper.SetDefault("LOG_MAX_BACKUPS", 3)
	viper.SetDefault("LOG_MAX_AGE_DAYS", 7)
	viper.SetDefault("LOG_COMPRESS", true)
	viper.SetDefault("OTEL_SERVICE_NAME", "comics_service")
	viper.SetDefault("OTEL_TRACES_SAMPLER_PERCENTAGE", 10)
	viper.SetDefault("OTEL_EXPORTER_SECURE", false)
}

func bindEnvKeys() {
	keys := []string{
		"ENVIRONMENT",
		"HTTP_ADDRESS",
		"HTTP_PORT",
		"GRPC_ADDRESS",
		"GRPC_PORT",
		"HOST_URL",
		"INIT_TIMEOUT",
		"SCRAPE_INTERVAL",
		"GOOGLE_CLIENT_ID",
		"GOOGLE_CLIENT_SECRET",
		"COMICS_DB_DRIVER",
		"COMICS_SQLITE_PATH",
		"COMICS_POSTGRES_URL",
		"PY_BACKEND_URL",
		"CORS_ALLOWED_ORIGINS",
		"HTTP_TLS_CERT_FILE",
		"HTTP_TLS_KEY_FILE",
		"DB_HOST",
		"DB_PORT",
		"DB_ADDR",
		"DB_USER",
		"DB_PASS",
		"DB_NAME",
		"DB_TABLE_USERS",
		"DB_TABLE_COMICS",
		"DB_MAX_POOL_SIZE",
		"DB_MIN_POOL_SIZE",
		"DB_MAX_CONN_IDLE_TIME",
		"DB_MAX_CONN_LIFE_TIME",
		"DB_CONN_TIMEOUT",
		"DB_BACKOFF_TIMEOUT",
		"ACCESS_TOKEN_EXPIRY_HOUR",
		"REFRESH_TOKEN_EXPIRY_HOUR",
		"ACCESS_TOKEN_SECRET",
		"REFRESH_TOKEN_SECRET",
		"LOG_LEVEL",
		"LOG_DEBUG_CONSOLE",
		"LOG_OUTPUT_FILE",
		"LOG_MAX_SIZE_MB",
		"LOG_MAX_BACKUPS",
		"LOG_MAX_AGE_DAYS",
		"LOG_COMPRESS",
		"OTEL_SERVICE_NAME",
		"OTEL_TRACES_SAMPLER_PERCENTAGE",
		"OTEL_EXPORTER_SECURE",
		"OTEL_EXPORTER_GRPC_ENDPOINT",
	}
	for _, key := range keys {
		if err := viper.BindEnv(key); err != nil {
			log.Warn().Err(err).Str("key", key).Msg("Failed to bind environment key")
		}
	}
}

func (env *Env) applyLocalTLSDefaults() {
	if env.AppEnv.IsProduction() {
		return
	}
	if strings.TrimSpace(env.HTTPTLSCertFile) != "" || strings.TrimSpace(env.HTTPTLSKeyFile) != "" {
		return
	}
	if !localPathExists(defaultTLSCert) {
		return
	}
	if !localPathExists(defaultTLSKey) {
		return
	}
	env.HTTPTLSCertFile = defaultTLSCert
	env.HTTPTLSKeyFile = defaultTLSKey
}

func (env *Env) HasHTTPSTLSFiles() bool {
	if strings.TrimSpace(env.HTTPTLSCertFile) == "" || strings.TrimSpace(env.HTTPTLSKeyFile) == "" {
		return false
	}
	if !localPathExists(env.HTTPTLSCertFile) {
		return false
	}
	if !localPathExists(env.HTTPTLSKeyFile) {
		return false
	}
	return true
}

func (env *Env) ServerScheme() string {
	if env.HasHTTPSTLSFiles() {
		return "https"
	}
	return "http"
}

func localPathExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join("..", path)); err == nil {
		return true
	}
	return false
}

// Validate checks required full-app runtime configuration.
func (env *Env) Validate() error {
	missing := []string{}
	if strings.TrimSpace(env.AddressHTTP) == "" {
		missing = append(missing, "HTTP_ADDRESS")
	}
	if strings.TrimSpace(env.PortHTTP) == "" {
		missing = append(missing, "HTTP_PORT")
	}
	if strings.TrimSpace(env.HostURL) == "" {
		missing = append(missing, "HOST_URL")
	}
	if env.DBConfig == nil || strings.TrimSpace(env.DBConfig.Addr) == "" {
		missing = append(missing, "DB_ADDR")
	}
	if env.DBConfig == nil || strings.TrimSpace(env.DBConfig.Name) == "" {
		missing = append(missing, "DB_NAME")
	}
	if env.JWTConfig == nil || strings.TrimSpace(env.JWTConfig.AccessTokenSecret) == "" {
		missing = append(missing, "ACCESS_TOKEN_SECRET")
	}
	if env.JWTConfig == nil || strings.TrimSpace(env.JWTConfig.RefreshTokenSecret) == "" {
		missing = append(missing, "REFRESH_TOKEN_SECRET")
	}
	switch strings.ToLower(strings.TrimSpace(env.ComicsDBDriver)) {
	case "", "sqlite":
		if strings.TrimSpace(env.ComicsSQLitePath) == "" {
			missing = append(missing, "COMICS_SQLITE_PATH")
		}
	case "postgres", "postgresql":
		if strings.TrimSpace(env.ComicsPostgresURL) == "" {
			missing = append(missing, "COMICS_POSTGRES_URL")
		}
	default:
		return fmt.Errorf("COMICS_DB_DRIVER should be sqlite or postgres, got %q", env.ComicsDBDriver)
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required env values: %s", strings.Join(missing, ", "))
	}
	return nil
}

// IsProduction environment check
func (e AppEnv) IsProduction() bool { return e == Production }

// IsDevelopment environment check
func (e AppEnv) IsDevelopment() bool { return e == Development }

// IsTesting environment check
func (e AppEnv) IsTesting() bool { return e == Testing }

// parseAppEnv safely parses environment variable and validates the value
func parseAppEnv(rawEnv string) AppEnv {
	switch strings.ToLower(strings.TrimSpace(rawEnv)) {
	case "production", "prod":
		return Production
	case "testing", "test":
		return Testing
	case "development", "dev", "":
		return Development
	default:
		log.Warn().
			Str("value", rawEnv).
			Str("default", string(defaultEnv)).
			Msg("Invalid environment value, using default")
		return defaultEnv
	}
}
