package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeRedactsPostgresURL(t *testing.T) {
	type config struct {
		ComicsPostgresURL string
		HostURL           string
	}

	got := Sanitize(config{
		ComicsPostgresURL: "postgresql://user:password@example.com/db",
		HostURL:           "localhost:8081",
	})

	fields, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected sanitized struct map, got %T", got)
	}
	if fields["ComicsPostgresURL"] == "postgresql://user:password@example.com/db" {
		t.Fatal("expected Postgres URL to be redacted")
	}
	if fields["HostURL"] != "localhost:8081" {
		t.Fatalf("expected HostURL to remain visible, got %v", fields["HostURL"])
	}
}

func TestApplyLocalTLSDefaultsUsesSharedCertificateInDevelopment(t *testing.T) {
	root := t.TempDir()
	tlsDir := filepath.Join(root, "tls")
	if err := os.MkdirAll(tlsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tlsDir, "comics.crt"), []byte("test-cert"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tlsDir, "comics.key"), []byte("test-key"), 0o600); err != nil {
		t.Fatal(err)
	}
	goServerDir := filepath.Join(root, "go_server")
	if err := os.Mkdir(goServerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(goServerDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(wd)
	}()

	env := &Env{AppEnv: Development}
	env.applyLocalTLSDefaults()

	if env.HTTPTLSCertFile != defaultTLSCert {
		t.Fatalf("expected local cert default %q, got %q", defaultTLSCert, env.HTTPTLSCertFile)
	}
	if env.HTTPTLSKeyFile != defaultTLSKey {
		t.Fatalf("expected local key default %q, got %q", defaultTLSKey, env.HTTPTLSKeyFile)
	}
	if env.ServerScheme() != "https" {
		t.Fatalf("expected https scheme, got %q", env.ServerScheme())
	}
}

func TestApplyLocalTLSDefaultsLeavesProductionOptIn(t *testing.T) {
	env := &Env{AppEnv: Production}
	env.applyLocalTLSDefaults()

	if env.HTTPTLSCertFile != "" || env.HTTPTLSKeyFile != "" {
		t.Fatalf("expected production TLS defaults to stay blank, got cert=%q key=%q",
			env.HTTPTLSCertFile,
			env.HTTPTLSKeyFile,
		)
	}
	if env.ServerScheme() != "http" {
		t.Fatalf("expected http scheme, got %q", env.ServerScheme())
	}
}
