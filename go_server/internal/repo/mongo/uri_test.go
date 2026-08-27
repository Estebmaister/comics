package mongo

import (
	"testing"

	"comics/internal/repo"
)

func TestBuildMongoURIWithoutCredentials(t *testing.T) {
	got := buildMongoURI(&repo.DBConfig{Addr: "mongodb://localhost:27017"})
	if got != "mongodb://localhost:27017" {
		t.Fatalf("unexpected uri %q", got)
	}
}

func TestBuildMongoURIWithLocalCredentials(t *testing.T) {
	got := buildMongoURI(&repo.DBConfig{
		Addr: "mongodb://localhost:27017",
		User: "esteb",
		Pass: "localdev",
	})
	want := "mongodb://esteb:localdev@localhost:27017/?authSource=admin"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBuildMongoURIWithAtlasSRVAddr(t *testing.T) {
	got := buildMongoURI(&repo.DBConfig{
		Addr: "mongodb+srv://sandbox.ux3yw.mongodb.net/",
		User: "esteb",
		Pass: "secret",
	})
	want := "mongodb+srv://esteb:secret@sandbox.ux3yw.mongodb.net/?retryWrites=true&w=majority&appName=Sandbox"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBuildMongoURIWithAtlasHost(t *testing.T) {
	got := buildMongoURI(&repo.DBConfig{
		Addr: "cluster.example.mongodb.net",
		User: "esteb",
		Pass: "secret",
	})
	want := "mongodb+srv://esteb:secret@cluster.example.mongodb.net/?retryWrites=true&w=majority&appName=Sandbox"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
