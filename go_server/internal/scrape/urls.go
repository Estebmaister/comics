package scrape

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	urlSwitchOnce sync.Once
	urlSwitch     map[string][]string
	urlSwitchErr  error
)

func urlSwitchCandidates() []string {
	if path := os.Getenv("COMICS_URL_SWITCH"); path != "" {
		return []string{path}
	}
	candidates := []string{
		filepath.Join("config", "url_switch.json"),
		filepath.Join("..", "config", "url_switch.json"),
		filepath.Join("..", "..", "config", "url_switch.json"),
		filepath.Join("..", "..", "..", "config", "url_switch.json"),
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
		candidates = append(
			[]string{filepath.Join(repoRoot, "config", "url_switch.json")},
			candidates...,
		)
	}
	return candidates
}

func findURLSwitchFile() string {
	for _, candidate := range urlSwitchCandidates() {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func loadURLSwitch() (map[string][]string, error) {
	urlSwitchOnce.Do(func() {
		urlSwitch = map[string][]string{}
		path := findURLSwitchFile()
		if path == "" {
			urlSwitchErr = fmt.Errorf(
				"url_switch.json not found (set COMICS_URL_SWITCH or add config/url_switch.json)",
			)
			return
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			urlSwitchErr = err
			return
		}
		urlSwitchErr = json.Unmarshal(raw, &urlSwitch)
	})
	return urlSwitch, urlSwitchErr
}

func publisherURLPairs() ([][2]string, error) {
	mappings, err := loadURLSwitch()
	if err != nil {
		return nil, err
	}
	active := []string{
		"ManhuaPlus",
		"Asura",
		"RealmScans",
		"DemonicScans",
		"Manganato",
	}
	pairs := make([][2]string, 0)
	for _, name := range active {
		for _, url := range mappings[name] {
			pairs = append(pairs, [2]string{name, url})
		}
	}
	return pairs, nil
}

func publisherBaseURL(publisherName string) string {
	mappings, err := loadURLSwitch()
	if err != nil {
		return ""
	}
	urls := mappings[publisherName]
	if len(urls) == 0 {
		return ""
	}
	return urls[0]
}

func publisherNameForID(id int) (string, error) {
	switch id {
	case PublisherAsura:
		return "Asura", nil
	case PublisherManhuaPlus:
		return "ManhuaPlus", nil
	case PublisherFlameScans:
		return "FlameScans", nil
	case PublisherRealmScans:
		return "RealmScans", nil
	case PublisherManganato:
		return "Manganato", nil
	case PublisherDemonicScans:
		return "DemonicScans", nil
	default:
		return "", fmt.Errorf("unknown publisher id %d", id)
	}
}
