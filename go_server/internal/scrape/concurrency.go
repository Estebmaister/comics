package scrape

import (
	"os"
	"strconv"
)

const defaultFetchConcurrency = 6

func fetchConcurrencyLimit() int {
	raw := os.Getenv("SCRAPE_FETCH_CONCURRENCY")
	if raw == "" {
		return defaultFetchConcurrency
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return defaultFetchConcurrency
	}
	return value
}
