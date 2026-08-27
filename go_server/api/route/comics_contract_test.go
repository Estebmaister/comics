package route

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"comics/domain"

	"github.com/gin-gonic/gin"
)

func TestComicsContractListSearchAndDuplicate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	comics := newTestComicService(t)
	comicsRouter(comics, router.Group("/"))

	ctx := context.Background()
	if _, err := comics.Create(ctx, domain.Comic{
		Titles: []string{"Contract hero"}, CurrentChap: 10, ComType: 3, Track: true, Rating: 4,
	}); err != nil {
		t.Fatal(err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/comics?from=0&limit=20&only_tracked=true&sort_by=rating&sort_dir=desc", nil)
	listRes := httptest.NewRecorder()
	router.ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("list expected 200, got %d", listRes.Code)
	}
	var listPayload []map[string]any
	if err := json.Unmarshal(listRes.Body.Bytes(), &listPayload); err != nil {
		t.Fatal(err)
	}
	if len(listPayload) < 1 {
		t.Fatalf("expected comics in list, got %d", len(listPayload))
	}

	searchReq := httptest.NewRequest(http.MethodGet, "/comics/search/contract?from=0&limit=20", nil)
	searchRes := httptest.NewRecorder()
	router.ServeHTTP(searchRes, searchReq)
	if searchRes.Code != http.StatusOK {
		t.Fatalf("search expected 200, got %d", searchRes.Code)
	}

	dupReq := httptest.NewRequest(http.MethodPost, "/comics", bytes.NewBufferString(`{"titles":["Contract hero"],"com_type":3}`))
	dupReq.Header.Set("Content-Type", "application/json")
	dupRes := httptest.NewRecorder()
	router.ServeHTTP(dupRes, dupReq)
	if dupRes.Code != http.StatusBadRequest {
		t.Fatalf("duplicate expected 400, got %d: %s", dupRes.Code, dupRes.Body.String())
	}
}

func TestComicsContractUpdateMergeDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	comics := newTestComicService(t)
	comicsRouter(comics, router.Group("/"))

	create := httptest.NewRequest(http.MethodPost, "/comics", bytes.NewBufferString(`{"titles":["Base hero"],"com_type":3}`))
	create.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	router.ServeHTTP(createRes, create)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create base expected 201, got %d", createRes.Code)
	}

	dupCreate := httptest.NewRequest(http.MethodPost, "/comics", bytes.NewBufferString(`{"titles":["Merge target"],"com_type":3,"current_chap":20,"cover":"cover.webp"}`))
	dupCreate.Header.Set("Content-Type", "application/json")
	dupCreateRes := httptest.NewRecorder()
	router.ServeHTTP(dupCreateRes, dupCreate)
	if dupCreateRes.Code != http.StatusCreated {
		t.Fatalf("create merge target expected 201, got %d", dupCreateRes.Code)
	}

	patchReq := httptest.NewRequest(http.MethodPut, "/comics/1", bytes.NewBufferString(`{"track":false,"rating":2,"viewed_chap":10}`))
	patchReq.Header.Set("Content-Type", "application/json")
	patchRes := httptest.NewRecorder()
	router.ServeHTTP(patchRes, patchReq)
	if patchRes.Code != http.StatusOK {
		t.Fatalf("update expected 200, got %d", patchRes.Code)
	}

	mergeReq := httptest.NewRequest(http.MethodPatch, "/comics/1/2", nil)
	mergeRes := httptest.NewRecorder()
	router.ServeHTTP(mergeRes, mergeReq)
	if mergeRes.Code != http.StatusOK {
		t.Fatalf("merge expected 200, got %d: %s", mergeRes.Code, mergeRes.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/comics/1", nil)
	deleteRes := httptest.NewRecorder()
	router.ServeHTTP(deleteRes, deleteReq)
	if deleteRes.Code != http.StatusAccepted && deleteRes.Code != http.StatusOK {
		t.Fatalf("delete expected 200/202, got %d", deleteRes.Code)
	}
}
