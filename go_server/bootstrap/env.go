package bootstrap

import (
	"context"
	"time"

	"comics/internal/logger"
	"comics/internal/repo"
	"comics/internal/tracing"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Enums for the app environment
type AppEnv string

const (
	Development AppEnv = "development"
	Production  AppEnv = "production"
	Testing     AppEnv = "testing"
)

// Env holds the application configuration
type Env struct {
	DB             repo.DBConfig
	JWT            JWTConfig
	Logger         logger.LoggerConfig
	Tracer         tracing.TracerConfig
	AppEnv         AppEnv `mapstructure:"ENVIRONMENT"`
	ServerAddress  string `mapstructure:"SERVER_ADDRESS"`
	GRPCAddress    string `mapstructure:"GRPC_ADDRESS"`
	ContextTimeout int    `mapstructure:"CONTEXT_TIMEOUT"`
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
	// viper.AutomaticEnv() // Read from environment variables
	env := Env{}
	jwtConfig := JWTConfig{}
	dbConfig := repo.DBConfig{}
	loggerConfig := logger.LoggerConfig{}
	tracerConfig := tracing.TracerConfig{}

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
	env.DB = dbConfig
	env.JWT = jwtConfig
	env.Logger = loggerConfig
	env.Tracer = tracerConfig
	env.DB.TracerConfig = env.Tracer

	// Cast the application environment to a type
	env.AppEnv = AppEnv(env.AppEnv)
	if env.AppEnv != Production {
		log.Debug().Msg("The App is running in a dev env")
		log.Debug().Msgf("%#v\n", env)
	}

	return &env
}
