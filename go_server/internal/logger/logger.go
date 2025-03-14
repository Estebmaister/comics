package logger

import (
	"context"
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// Default output for debug console
	output io.Writer = zerolog.NewConsoleWriter(
		func(w *zerolog.ConsoleWriter) {
			w.FieldsExclude = []string{
				"span_id", "trace_id", "request_id", "client_ip"}
		})
)

// Set the default logger to console in case of errors previous to read the env
func init() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	logger := zerolog.New(output).With().Timestamp().Caller().Logger()
	log.Logger = logger
}

// LoggerConfig holds the configuration for the logger
type LoggerConfig struct {
	LogLevel        string `mapstructure:"LOG_LEVEL" default:"info"`
	LogDebugConsole bool   `mapstructure:"LOG_DEBUG_CONSOLE"`
	LogOutputFile   string `mapstructure:"LOG_OUTPUT_FILE"`

	MaxSize    int  `mapstructure:"LOG_MAX_SIZE_MB"`
	MaxBackups int  `mapstructure:"LOG_MAX_BACKUPS"`
	MaxAge     int  `mapstructure:"LOG_MAX_AGE_DAYS"`
	Compress   bool `mapstructure:"LOG_COMPRESS"`
}

type shutdownFunc func(context.Context) error

func InitLogger(ctx context.Context, cfg *LoggerConfig) (zerolog.Logger, shutdownFunc, error) {
	if cfg == nil {
		cfg = &LoggerConfig{}
	}
	if !cfg.LogDebugConsole { // change default output to plain console
		output = os.Stderr
	}

	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}

	// Configure lumberjack for log rotation
	lumberjackLogger := &lumberjack.Logger{
		Filename:   cfg.LogOutputFile,
		MaxSize:    cfg.MaxSize,    // Megabytes
		MaxBackups: cfg.MaxBackups, // Number of backups
		MaxAge:     cfg.MaxAge,     // Days
		Compress:   cfg.Compress,   // Compress backups
	}

	// Create a multi-writer to write to both console and file
	multi := zerolog.MultiLevelWriter(output, lumberjackLogger)
	// Initialize the logger
	logger := zerolog.New(multi).With().Timestamp().Logger()

	// Set the global logger
	zerolog.SetGlobalLevel(level)
	zerolog.DefaultContextLogger = &logger
	log.Logger = logger

	return logger, func(_ context.Context) error { return lumberjackLogger.Close() }, nil
}
