package bootstrap

import (
	"context"
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

const (
	Development AppEnv = "development"
	Production  AppEnv = "production"
	Testing     AppEnv = "testing"

	// Default environment if not specified
	defaultEnv = Production

	defaultCtxTimeout = 10 * time.Second
)

// Env holds the application configuration
type Env struct {
	*repo.DBConfig
	*JWTConfig
	*logger.LoggerConfig
	*tracer.TracerConfig
	AppEnv         `mapstructure:"ENVIRONMENT"`
	AddressHTTP    string        `mapstructure:"ADDRESS_HTTP"`
	AddressGRPC    string        `mapstructure:"ADDRESS_GRPC"`
	InitCtxTimeout time.Duration `mapstructure:"INIT_TIMEOUT"`
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
	env := Env{}
	jwtConfig := &JWTConfig{}
	dbConfig := &repo.DBConfig{}
	loggerConfig := &logger.LoggerConfig{}
	tracerConfig := &tracer.TracerConfig{}

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Can't find the file .env")
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
	if !env.AppEnv.IsProduction() {
		log.Debug().Msg("The App is running in a dev env")
		log.Debug().Msgf("%v\n", Sanitize(env))
	}

	// Set the default initialization timeout
	if env.InitCtxTimeout == 0 {
		env.InitCtxTimeout = defaultCtxTimeout
	}

	return &env
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
