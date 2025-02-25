package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/prometheus/common/model"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/grafana/loki-client-go/loki"
	"github.com/grafana/loki-client-go/pkg/urlutil"
)

var (
	output = zerolog.NewConsoleWriter()
)

func init() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	logger := zerolog.New(output).With().Timestamp().Caller().Logger()
	log.Logger = logger
}

// LoggerConfig holds the configuration for the logger
type LoggerConfig struct {
	LogLevel      string `mapstructure:"LOG_LEVEL" default:"info"`
	LogFormat     string `mapstructure:"LOG_FORMAT" default:"json"`
	LogOutputFile string `mapstructure:"LOG_OUTPUT_FILE"`

	MaxSize    int  `mapstructure:"LOG_MAX_SIZE_MB"`
	MaxBackups int  `mapstructure:"LOG_MAX_BACKUPS"`
	MaxAge     int  `mapstructure:"LOG_MAX_AGE_DAYS"`
	Compress   bool `mapstructure:"LOG_COMPRESS"`

	LokiEndpoint string `mapstructure:"LOKI_ENDPOINT"`
}

func InitLogger(cfg LoggerConfig) (zerolog.Logger, func(), error) {
	if cfg.LogFormat == "json" {
		output = zerolog.ConsoleWriter{Out: os.Stdout, NoColor: true}
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
	lokiURLValue, err := url.Parse(cfg.LokiEndpoint)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse Loki endpoint")
		return logger, func() { lumberjackLogger.Close() }, err
	}
	client, err := loki.New(loki.Config{URL: urlutil.URLValue{URL: lokiURLValue}})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create Loki client")
		return logger, func() { lumberjackLogger.Close() }, err
	}

	// Channel for log messages
	logChannel := make(chan string, 100)

	// WaitGroup to wait for goroutine to finish
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	// Goroutine to process log messages
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-logChannel:
				// Prepare the payload for Loki.
				payload := map[string]interface{}{
					"streams": []map[string]interface{}{
						{
							"labels": `{app="my-go-app", environment="production"}`,
							"entries": []map[string]interface{}{
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
	shutdown := func() {
		// Cancel the context to stop the goroutine
		cancel()
		// Wait for the goroutine to finish
		wg.Wait()
		// Close the Loki client
		client.Stop()
		// Close the lumberjack logger
		lumberjackLogger.Close()
		log.Info().Msg("Logger shutdown")
	}

	return logger, shutdown, nil
}

// sendToLoki sends log data to Loki with simple retry logic.
func sendToLoki(client *loki.Client, logData map[string]interface{}) error {
	// Convert logData to JSON string
	jsonData, err := json.Marshal(logData)
	if err != nil {
		return fmt.Errorf("failed to marshal log data: %w", err)
	}

	retries := 3
	for i := 0; i < retries; i++ {
		err := client.Handle(model.LabelSet{
			model.LabelName("app"):         model.LabelValue("my-go-app"),
			model.LabelName("environment"): model.LabelValue("production"),
		}, time.Now(), string(jsonData))
		if err == nil {
			return nil // Successfully sent.
		}
		time.Sleep(2 * time.Second) // Wait before retrying.
	}
	return fmt.Errorf("failed to send log to Loki after %d retries", retries)
}
