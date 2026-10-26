package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	comicrepo "comics/internal/repo/comics"
)

func TestAuditLiveDiscoveryIntegrity(t *testing.T) {
	if os.Getenv("AUDIT_DISCOVERY") != "1" {
		t.Skip("set AUDIT_DISCOVERY=1 to inspect the live catalog read-only")
	}
	path := os.Getenv("COMICS_SQLITE_PATH")
	if path == "" {
		path = "../../../db/comics.db"
	}
	before := fileDigestSet(t, path)
	repo, err := comicrepo.NewSQLiteComicRepositoryReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	audit, err := repo.AuditDiscoveryIntegrity(context.Background())
	closeErr := repo.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	after := fileDigestSet(t, path)
	if diff := cmp.Diff(before, after); diff != "" {
		t.Fatalf("audit changed SQLite files (-before +after):\n%s", diff)
	}
	t.Logf(
		"total=%d active=%d keys=%d keyless=%d conflicts=%d restorable=%d blocked=%d deleted_keys=%d orphan_keys=%d",
		audit.Summary.TotalComics,
		audit.Summary.ActiveComics,
		audit.Summary.KeyRows,
		audit.Summary.KeylessActive,
		audit.Summary.ActiveConflicts,
		audit.Summary.Restorable,
		audit.Summary.BlockedNoKey,
		audit.Summary.DeletedKeyRows,
		audit.Summary.OrphanKeyRows,
	)
}

type fileDigest struct {
	Mode    os.FileMode
	Size    int64
	ModTime int64
	Hash    string
}

func fileDigestSet(t *testing.T, dbPath string) map[string]fileDigest {
	t.Helper()
	paths := []string{dbPath}
	for _, suffix := range []string{"-wal", "-shm"} {
		candidate := dbPath + suffix
		if _, err := os.Stat(candidate); err == nil {
			paths = append(paths, candidate)
		}
	}
	out := map[string]fileDigest{}
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.New()
		if _, err := io.Copy(hash, file); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
		_ = file.Close()
		abs, _ := filepath.Abs(path)
		out[abs] = fileDigest{
			Mode:    info.Mode(),
			Size:    info.Size(),
			ModTime: info.ModTime().UnixNano(),
			Hash:    hex.EncodeToString(hash.Sum(nil)),
		}
	}
	return out
}
