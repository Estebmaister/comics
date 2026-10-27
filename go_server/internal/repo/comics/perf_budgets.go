package comics

import (
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// PerfBudgets holds opt-in strict performance gates (local snapshot).
type PerfBudgets struct {
	FeedbackMS              int `yaml:"feedback_ms"`
	CachedPagePaintMS       int `yaml:"cached_page_paint_ms"`
	ListHTTPP95MS           int `yaml:"list_http_p95_ms"`
	SearchHTTPP95MS         int `yaml:"search_http_p95_ms"`
	SearchHTTPP99MS         int `yaml:"search_http_p99_ms"`
	SearchTrigramP95MS      int `yaml:"search_trigram_p95_ms"`
	SearchTrigramP99MS      int `yaml:"search_trigram_p99_ms"`
	SearchShortP95MS          int `yaml:"search_short_p95_ms"`
	SearchUnderScrapeP95MS  int `yaml:"search_under_scrape_p95_ms"`
	SearchUnderScrapeP99MS  int `yaml:"search_under_scrape_p99_ms"`
	ReadDuringWriteP95MS    int `yaml:"read_during_write_p95_ms"`
}

func LoadPerfBudgets(path string) (PerfBudgets, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return PerfBudgets{}, err
	}
	var budgets PerfBudgets
	if err := yaml.Unmarshal(raw, &budgets); err != nil {
		return PerfBudgets{}, err
	}
	return budgets, nil
}

func PerfStrictEnabled() bool {
	return os.Getenv("COMICS_PERF_STRICT") == "1"
}

// PerfBudgetsPath resolves the YAML budgets file for strict perf gates.
func PerfBudgetsPath() string {
	return perfBudgetsPath()
}

func perfBudgetsPath() string {
	if path := os.Getenv("COMICS_PERF_BUDGETS"); path != "" {
		return path
	}
	for _, candidate := range []string{
		"testdata/perf/budgets.yaml",
		"../testdata/perf/budgets.yaml",
	} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// PerfBudgetMS resolves a budget entry with optional COMICS_PERF_* env overrides.
func PerfBudgetMS(budgets PerfBudgets, key string, fallback int) int {
	return perfBudgetMS(budgets, key, fallback)
}

func perfBudgetMS(budgets PerfBudgets, key string, fallback int) int {
	switch key {
	case "search_trigram_p95":
		if budgets.SearchTrigramP95MS > 0 {
			return budgets.SearchTrigramP95MS
		}
	case "search_trigram_p99":
		if budgets.SearchTrigramP99MS > 0 {
			return budgets.SearchTrigramP99MS
		}
	case "search_short_p95":
		if budgets.SearchShortP95MS > 0 {
			return budgets.SearchShortP95MS
		}
	case "search_under_scrape_p95":
		if budgets.SearchUnderScrapeP95MS > 0 {
			return budgets.SearchUnderScrapeP95MS
		}
	case "search_under_scrape_p99":
		if budgets.SearchUnderScrapeP99MS > 0 {
			return budgets.SearchUnderScrapeP99MS
		}
	case "list_http_p95":
		if budgets.ListHTTPP95MS > 0 {
			return budgets.ListHTTPP95MS
		}
	case "search_http_p95":
		if budgets.SearchHTTPP95MS > 0 {
			return budgets.SearchHTTPP95MS
		}
	case "search_http_p99":
		if budgets.SearchHTTPP99MS > 0 {
			return budgets.SearchHTTPP99MS
		}
	}
	if v := os.Getenv("COMICS_PERF_" + key); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			return ms
		}
	}
	return fallback
}
