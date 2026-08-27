package scrape

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed url_switch.json
var urlSwitchRaw []byte

var (
	urlSwitchOnce sync.Once
	urlSwitch     map[string][]string
	urlSwitchErr  error
)

func loadURLSwitch() (map[string][]string, error) {
	urlSwitchOnce.Do(func() {
		urlSwitch = map[string][]string{}
		urlSwitchErr = json.Unmarshal(urlSwitchRaw, &urlSwitch)
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
		"FlameScans",
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
