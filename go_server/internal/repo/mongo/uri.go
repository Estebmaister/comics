package mongo

import (
	"fmt"
	"net/url"
	"strings"

	"comics/internal/repo"
)

func buildMongoURI(cfg *repo.DBConfig) string {
	addr := strings.TrimSpace(cfg.Addr)
	if addr == "" {
		addr = "mongodb://localhost:27017"
	}

	user := strings.TrimSpace(cfg.User)
	pass := cfg.Pass
	if user == "" && pass == "" {
		return addr
	}

	escapedUser := url.QueryEscape(user)
	escapedPass := url.QueryEscape(pass)

	if strings.HasPrefix(addr, "mongodb+srv://") {
		host := strings.TrimPrefix(addr, "mongodb+srv://")
		host = strings.TrimSuffix(strings.Split(host, "/")[0], "/")
		host = strings.Split(host, "?")[0]
		return fmt.Sprintf(
			"mongodb+srv://%s:%s@%s/?retryWrites=true&w=majority&appName=Sandbox",
			escapedUser,
			escapedPass,
			host,
		)
	}

	if strings.HasPrefix(addr, "mongodb://") {
		host := strings.TrimPrefix(addr, "mongodb://")
		host = strings.TrimSuffix(strings.Split(host, "/")[0], "/")
		host = strings.Split(host, "?")[0]
		return fmt.Sprintf(
			"mongodb://%s:%s@%s/?authSource=admin",
			escapedUser,
			escapedPass,
			host,
		)
	}

	return fmt.Sprintf(
		"mongodb+srv://%s:%s@%s/?retryWrites=true&w=majority&appName=Sandbox",
		escapedUser,
		escapedPass,
		addr,
	)
}
