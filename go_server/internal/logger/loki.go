package logger

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// // Configure the Loki writer
// lokiEndpoint := "http://localhost:3100/api/prom/push"
// labels := `{app="my-go-app", environment="production"}`
// lokiWriter := NewLokiWriter(lokiEndpoint, labels)

// // Initialize zerolog with the Loki writer
// logger := zerolog.New(lokiWriter).With().Timestamp().Logger()

type LokiWriter struct {
	endpoint string
	labels   string
	client   *http.Client
}

func NewLokiWriter(endpoint, labels string) *LokiWriter {
	return &LokiWriter{
		endpoint: endpoint,
		labels:   labels,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (lw *LokiWriter) Write(p []byte) (n int, err error) {
	var event map[string]interface{}
	if err := json.Unmarshal(p, &event); err != nil {
		return 0, err
	}

	timestamp := time.Now().UnixNano()
	line, err := json.Marshal(event)
	if err != nil {
		return 0, err
	}

	payload := map[string]interface{}{
		"streams": []map[string]interface{}{
			{
				"labels": lw.labels,
				"entries": []map[string]interface{}{
					{
						"ts":   timestamp,
						"line": string(line),
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest("POST", lw.endpoint, bytes.NewBuffer(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := lw.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return 0, err
	}

	return len(p), nil
}
