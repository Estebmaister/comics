package route

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"comics/api/middleware"
	"comics/internal/tokenutil"
)

func TestCatalogReadRoutesRemainPublic(t *testing.T) {
	router := newComicTestRouter(newTestComicService(t))

	tests := []struct {
		path     string
		wantCode int
	}{
		{path: "/comics?from=0&limit=1", wantCode: http.StatusOK},
		{path: "/comics/search/missing?from=0&limit=1", wantCode: http.StatusOK},
		{path: "/comics/999999999", wantCode: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.Header.Set(middleware.KeyAccept, middleware.ContentTypeJSON)
			res := httptest.NewRecorder()

			router.ServeHTTP(res, req)

			if res.Code != tt.wantCode {
				t.Fatalf("%s: expected %d, got %d: %s", tt.path, tt.wantCode, res.Code, res.Body.String())
			}
		})
	}
}

func TestCatalogAdminRoutesRejectAnonymousUserAndCookieOnly(t *testing.T) {
	routes := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "create", method: http.MethodPost, path: "/comics", body: `{"titles":["Denied create"]}`},
		{name: "update", method: http.MethodPut, path: "/comics/1", body: `{"track":true}`},
		{name: "delete", method: http.MethodDelete, path: "/comics/1"},
		{name: "cover", method: http.MethodPatch, path: "/comics/1/cover-visibility", body: `{"cover":"cover.webp","cover_visible":false}`},
		{name: "merge patch", method: http.MethodPatch, path: "/comics/1/2"},
		{name: "merge put", method: http.MethodPut, path: "/comics/1/2"},
		{name: "scrape", method: http.MethodGet, path: "/scrape"},
		{name: "scrape status", method: http.MethodGet, path: "/scrape/status"},
	}

	principals := []struct {
		name      string
		configure func(*testing.T, *http.Request)
		wantCode  int
	}{
		{
			name:      "anonymous",
			configure: func(*testing.T, *http.Request) {},
			wantCode:  http.StatusUnauthorized,
		},
		{
			name: "admin cookie without bearer",
			configure: func(t *testing.T, req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  middleware.KeyAccessToken,
					Value: catalogTokenForRole(t, tokenutil.RoleAdmin),
				})
			},
			wantCode: http.StatusUnauthorized,
		},
		{
			name: "user bearer",
			configure: func(t *testing.T, req *http.Request) {
				authorizeCatalogRequest(t, req, tokenutil.RoleUser)
			},
			wantCode: http.StatusForbidden,
		},
	}

	for _, route := range routes {
		for _, principal := range principals {
			t.Run(route.name+"/"+principal.name, func(t *testing.T) {
				service := &scrapeStubComicService{ComicUseCase: newTestComicService(t)}
				router := newComicTestRouter(service)
				req := httptest.NewRequest(route.method, route.path, bytes.NewBufferString(route.body))
				req.Header.Set(middleware.KeyAccept, middleware.ContentTypeJSON)
				if route.body != "" {
					req.Header.Set("Content-Type", "application/json")
				}
				principal.configure(t, req)
				res := httptest.NewRecorder()

				router.ServeHTTP(res, req)

				if res.Code != principal.wantCode {
					t.Fatalf(
						"%s %s as %s: expected %d, got %d: %s",
						route.method,
						route.path,
						principal.name,
						principal.wantCode,
						res.Code,
						res.Body.String(),
					)
				}
			})
		}
	}
}
