package route

import (
	"bytes"
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"comics/domain"
	comicrepo "comics/internal/repo/comics"
	"comics/internal/usecase"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

func newTestComicService(t *testing.T) domain.ComicUseCase {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "comics.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE comics (
			id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
			titles TEXT NOT NULL,
			current_chap INTEGER NOT NULL DEFAULT 0,
			cover TEXT NOT NULL DEFAULT '',
			last_update INTEGER NOT NULL,
			com_type INTEGER NOT NULL DEFAULT 0,
			status INTEGER NOT NULL DEFAULT 0,
			published_in TEXT NOT NULL DEFAULT '0',
			genres TEXT NOT NULL DEFAULT '0',
			description TEXT NOT NULL DEFAULT '',
			author TEXT NOT NULL DEFAULT '',
			track BOOLEAN NOT NULL DEFAULT 0,
			viewed_chap INTEGER NOT NULL DEFAULT 0,
			rating INTEGER NOT NULL DEFAULT 0,
			deleted BOOLEAN NOT NULL DEFAULT 0,
			cover_visible BOOLEAN NOT NULL DEFAULT 1,
			identity_key TEXT NOT NULL DEFAULT ''
		)
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	repo, err := comicrepo.NewSQLiteComicRepository(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	comics := usecase.NewComicService(repo, nil)
	t.Cleanup(func() { _ = comics.Close() })
	return comics
}

func TestComicsRouterRejectsInvalidPathID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	comicsRouter(newTestComicService(t), router.Group("/"))

	req := httptest.NewRequest(http.MethodGet, "/comics/not-a-number", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", res.Code, res.Body.String())
	}
}

func TestComicsRouterRejectsInvalidPaginationQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	comicsRouter(newTestComicService(t), router.Group("/"))

	for _, path := range []string{
		"/comics?from=bad",
		"/comics?limit=bad",
		"/comics/search/sample?from=bad",
		"/comics/search/sample?limit=bad",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)

		if res.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d: %s", path, res.Code, res.Body.String())
		}
	}
}

func TestComicsRouterRejectsExplicitEmptyTitleUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	comicsRouter(newTestComicService(t), router.Group("/"))

	createReq := httptest.NewRequest(
		http.MethodPost,
		"/comics",
		bytes.NewBufferString(`{"titles":["Original"]}`),
	)
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	router.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", createRes.Code, createRes.Body.String())
	}

	updateReq := httptest.NewRequest(
		http.MethodPut,
		"/comics/1",
		bytes.NewBufferString(`{"titles":[]}`),
	)
	updateReq.Header.Set("Content-Type", "application/json")
	updateRes := httptest.NewRecorder()
	router.ServeHTTP(updateRes, updateReq)
	if updateRes.Code != http.StatusBadRequest {
		t.Fatalf("expected update 400, got %d: %s", updateRes.Code, updateRes.Body.String())
	}
}

func TestComicsRouterRejectsDuplicateCreate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	comicsRouter(newTestComicService(t), router.Group("/"))

	createReq := httptest.NewRequest(
		http.MethodPost,
		"/comics",
		bytes.NewBufferString(`{"titles":["Duplicate hero"],"com_type":3}`),
	)
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	router.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", createRes.Code, createRes.Body.String())
	}

	dupReq := httptest.NewRequest(
		http.MethodPost,
		"/comics",
		bytes.NewBufferString(`{"titles":["duplicate hero"],"com_type":3}`),
	)
	dupReq.Header.Set("Content-Type", "application/json")
	dupRes := httptest.NewRecorder()
	router.ServeHTTP(dupRes, dupReq)
	if dupRes.Code != http.StatusBadRequest {
		t.Fatalf("expected duplicate 400, got %d: %s", dupRes.Code, dupRes.Body.String())
	}
}

func TestScrapeReturnsSuccessWithMockRunner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc := &scrapeStubComicService{ComicUseCase: newTestComicService(t)}
	comicsRouter(svc, router.Group("/"))

	req := httptest.NewRequest(http.MethodGet, "/scrape", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
}

type scrapeStubComicService struct {
	domain.ComicUseCase
}

func (s *scrapeStubComicService) Scrape(context.Context) error {
	return nil
}

func (s *scrapeStubComicService) ScrapeStatus(context.Context) (domain.ScrapeStatus, error) {
	return domain.ScrapeStatus{}, nil
}

func TestScrapeStatusEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc := newTestComicService(t)
	comicsRouter(svc, router.Group("/"))

	req := httptest.NewRequest(http.MethodGet, "/scrape/status", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
}

func TestCORSMiddlewareAllowsConfiguredOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(corsMiddleware("http://localhost:3000"))
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

	req := httptest.NewRequest(http.MethodOptions, "/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", res.Code, res.Body.String())
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("expected CORS origin header, got %q", got)
	}
	if got := res.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected credentials CORS header, got %q", got)
	}
}
