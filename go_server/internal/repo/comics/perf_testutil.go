package comics

import (
	"os"
	"path/filepath"
	"strings"
)

// perfCatalogDBPath returns a verified SQLite snapshot for opt-in perf benchmarks.
// Set COMICS_PERF_DB to an absolute or repo-relative path; never point at a live writer DB under test.
// PerfCatalogDBPath resolves the opt-in catalog performance snapshot path.
func PerfCatalogDBPath() string {
	if path := os.Getenv("COMICS_PERF_DB"); path != "" {
		return resolvePerfCatalogDBPath(path)
	}
	for _, candidate := range []string{
		filepath.Join("..", "db", "comics.db"),
		filepath.Join("..", "..", "..", "db", "comics.db"),
		filepath.Join("db", "comics.db"),
	} {
		if resolved := resolvePerfCatalogDBPath(candidate); resolved != "" {
			return resolved
		}
	}
	return ""
}

func resolvePerfCatalogDBPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	candidates := []string{path}
	if !filepath.IsAbs(path) {
		candidates = append(candidates, filepath.Join("..", path))
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err != nil {
			continue
		}
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		return abs
	}
	return ""
}
