package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/prometheus/common/model"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/grafana/loki-client-go/loki"
)

var (
	// Default output for debug console
	output io.Writer = zerolog.NewConsoleWriter(
		func(w *zerolog.ConsoleWriter) {
			w.FieldsExclude = []string{
				"span_id", "trace_id", "request_id", "client_ip"}
		})

	// Use structured labels instead of raw string
	labels = model.LabelSet{
		"app":         "comics",
		"environment": "production",
	}
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

	LokiEndpoint string `mapstructure:"LOKI_ENDPOINT"`
}

func InitLogger(ctx context.Context, cfg *LoggerConfig) (
	zerolog.Logger, func(ctx context.Context) error, error) {
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

	// Loki client setup
	client, err := loki.NewWithDefault(cfg.LokiEndpoint)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create Loki client")
		return logger, func(_ context.Context) error { return lumberjackLogger.Close() }, err
	}

	// Channel for log messages & WaitGroup to wait for goroutine to finish
	logChannel := make(chan string, 100)
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)

	// Goroutine to process log messages
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-logChannel:
				// Prepare the payload for Loki
				payload := map[string]any{
					"streams": []map[string]any{
						{
							"labels": labels,
							"entries": []map[string]any{
								{
									"ts":   time.Now().Format(time.RFC3339Nano),
									"line": msg,
								},
							},
						},
					},
				}

				// Send log to Loki with retry logic.
				if err := sendToLoki(client, payload); err != nil {
					log.Error().Err(err).Msg("Failed to send log to Loki")
				}
			}
		}

	}()

	// Function to gracefully shut down the logger
	shutdown := func(_ context.Context) error {
		// Cancel the context to stop the goroutine
		cancel()
		// Wait for the goroutine to finish
		wg.Wait()
		// Close the Loki client
		client.Stop()
		// Close the lumberjack logger
		err := lumberjackLogger.Close()
		if err != nil {
			log.Error().Err(err).Msg("Failed to close lumberjack logger")
		}
		// Close the channel
		close(logChannel)
		log.Info().Msg("Logger shutdown successfull")
		return nil
	}

	return logger, shutdown, nil
}

// sendToLoki sends log data to Loki with simple retry logic.
func sendToLoki(client *loki.Client, logData map[string]any) error {
	// Convert logData to JSON string
	jsonData, err := json.Marshal(logData)
	if err != nil {
		return fmt.Errorf("failed to marshal log data: %w", err)
	}

	retries := 3
	for i := range retries {
		err := client.Handle(labels, time.Now(), string(jsonData))
		if err == nil {
			return nil // Successfully sent.
		}
		backoff := time.Duration(2*i) * time.Second
		time.Sleep(backoff)
	}
	return fmt.Errorf("failed to send log to Loki after %d retries", retries)
}
