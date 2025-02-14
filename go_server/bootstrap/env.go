package bootstrap

import (
	"context"
	"log"
	"time"

	"github.com/spf13/viper"
)

// Env holds the application configuration
type Env struct {
	DB             DBConfig
	JWT            JWTConfig
	AppEnv         string `mapstructure:"APP_ENV"`
	ServerAddress  string `mapstructure:"SERVER_ADDRESS"`
	ContextTimeout int    `mapstructure:"CONTEXT_TIMEOUT"`
}

// JWTConfig holds the configuration for the JW Token
type JWTConfig struct {
	AccessTokenExpiryHour  time.Duration `mapstructure:"ACCESS_TOKEN_EXPIRY_HOUR"`
	RefreshTokenExpiryHour time.Duration `mapstructure:"REFRESH_TOKEN_EXPIRY_HOUR"`
	AccessTokenSecret      string        `mapstructure:"ACCESS_TOKEN_SECRET"`
	RefreshTokenSecret     string        `mapstructure:"REFRESH_TOKEN_SECRET"`
}

// DBConfig holds the DB configuration
type DBConfig struct {
	Addr       string `mapstructure:"DB_ADDR"`
	User       string `mapstructure:"DB_USER"`
	Pass       string `mapstructure:"DB_PASS"`
	Name       string `mapstructure:"DB_NAME"`
	TableUsers string `mapstructure:"DB_TABLE_USERS"`
}

// MustLoadEnv reads the environment configuration with viper
func MustLoadEnv(_ context.Context) *Env {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env") // Define the config type as ENV
	// viper.AutomaticEnv() // Read from environment variables
	env := Env{}
	dbConfig := DBConfig{}
	jwtConfig := JWTConfig{}

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal("Can't find the file .env : ", err)
	}

	err1 := viper.Unmarshal(&env)
	err2 := viper.Unmarshal(&jwtConfig)
	err3 := viper.Unmarshal(&dbConfig)
	if err1 != nil || err2 != nil || err3 != nil {
		log.Fatal("Environment can't be loaded: ", err)
	}
	env.DB = dbConfig
	env.JWT = jwtConfig

	if env.AppEnv == "development" {
		log.Println("The App is running in development env")
		log.Printf("%#v\n", env) // Debug
	}

	return &env
}
