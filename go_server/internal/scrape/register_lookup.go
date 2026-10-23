package scrape

import (
	"strings"

	"comics/internal/identity"
)

func mergeTitleVariants(existing []string, incoming []string, comType int) []string {
	seen := map[string]struct{}{}
	combined := make([]string, 0, len(existing)+len(incoming))
	for _, title := range append(existing, incoming...) {
		key := strings.ToLower(strings.TrimSpace(title))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		combined = append(combined, title)
	}
	return identity.NormalizeTitleVariants(combined, comType)
}
