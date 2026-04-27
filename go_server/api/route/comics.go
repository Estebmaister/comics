package route

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"comics/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func comicsRouter(group *gin.RouterGroup) {
	comics, err := service.NewSQLiteComicService(os.Getenv("COMICS_SQLITE_PATH"))
	if err != nil {
		log.Warn().Err(err).Msg("Comic REST routes disabled")
		return
	}

	group.GET("/health/db", func(c *gin.Context) {
		if err := comics.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "database unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	group.GET("/scrape", proxyPythonScrape)

	group.GET("/comics", listComics(comics))
	group.POST("/comics", createComic(comics))
	group.GET("/comics/:id", getComic(comics))
	group.PUT("/comics/:id", updateComic(comics))
	group.DELETE("/comics/:id", deleteComic(comics))
	group.PATCH("/comics/:id/cover-visibility", updateCoverVisibility(comics))
	group.GET("/comics/search/:title", searchComics(comics))
	group.PATCH("/comics/:base_id/:merging_id", mergeComics(comics))
	group.PUT("/comics/:base_id/:merging_id", mergeComics(comics))
}

func listComics(comics *service.SQLiteComicService) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := comics.List(
			c.Request.Context(),
			queryInt(c, "from", 0),
			queryInt(c, "limit", 20),
			queryBool(c, "only_tracked"),
			queryBool(c, "only_unchecked"),
			queryBool(c, "full"),
		)
		writeComicList(c, result, err)
	}
}

func searchComics(comics *service.SQLiteComicService) gin.HandlerFunc {
	return func(c *gin.Context) {
		title := strings.TrimSpace(c.Param("title"))
		if title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Title cannot be empty"})
			return
		}
		result, err := comics.Search(
			c.Request.Context(),
			title,
			queryInt(c, "from", 0),
			queryInt(c, "limit", 20),
			queryBool(c, "only_tracked"),
			queryBool(c, "only_unchecked"),
			queryBool(c, "full"),
		)
		writeComicList(c, result, err)
	}
}

func getComic(comics *service.SQLiteComicService) gin.HandlerFunc {
	return func(c *gin.Context) {
		comic, err := comics.Get(c.Request.Context(), pathInt(c, "id"))
		writeComic(c, comic, err)
	}
}

func createComic(comics *service.SQLiteComicService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body map[string]any
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Body payload is necessary"})
			return
		}
		comic := service.ComicJSON{CoverVisible: true}
		applyComicPatch(&comic, body)
		if len(comic.Titles) == 0 || comic.Titles[0] == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "titles should be a non-empty list of strings"})
			return
		}
		created, err := comics.Create(c.Request.Context(), comic)
		writeComicWithStatus(c, created, err, http.StatusCreated)
	}
}

func updateComic(comics *service.SQLiteComicService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body map[string]any
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Body payload is necessary"})
			return
		}
		id := pathInt(c, "id")
		current, err := comics.Get(c.Request.Context(), id)
		if err != nil {
			writeComic(c, current, err)
			return
		}
		applyComicPatch(&current, body)
		comic, err := comics.Update(c.Request.Context(), id, current)
		writeComic(c, comic, err)
	}
}

func deleteComic(comics *service.SQLiteComicService) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := comics.Delete(c.Request.Context(), pathInt(c, "id"))
		if errors.Is(err, service.ErrComicNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "Comic not found"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusAccepted, http.StatusAccepted)
	}
}

func updateCoverVisibility(comics *service.SQLiteComicService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Cover        string `json:"cover"`
			CoverVisible *bool  `json:"cover_visible"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Cover == "" || body.CoverVisible == nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "cover and cover_visible are required"})
			return
		}
		comic, err := comics.UpdateCoverVisibility(
			c.Request.Context(),
			pathInt(c, "id"),
			body.Cover,
			*body.CoverVisible,
		)
		writeComic(c, comic, err)
	}
}

func mergeComics(comics *service.SQLiteComicService) gin.HandlerFunc {
	return func(c *gin.Context) {
		comic, err := comics.Merge(
			c.Request.Context(),
			pathInt(c, "base_id"),
			pathInt(c, "merging_id"),
		)
		if err != nil && strings.Contains(err.Error(), "same type") {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		writeComic(c, comic, err)
	}
}

func writeComicList(c *gin.Context, result service.ComicListResult, err error) {
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.Header("access-control-expose-headers", "total-comics,total-pages,current-page")
	c.Header("total-comics", strconv.Itoa(result.Total))
	c.Header("total-pages", strconv.Itoa(result.TotalPages))
	c.Header("current-page", strconv.Itoa(result.CurrentPage))
	c.JSON(http.StatusOK, result.Comics)
}

func writeComic(c *gin.Context, comic service.ComicJSON, err error) {
	writeComicWithStatus(c, comic, err, http.StatusOK)
}

func writeComicWithStatus(c *gin.Context, comic service.ComicJSON, err error, status int) {
	if errors.Is(err, service.ErrComicNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"message": "Comic not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(status, comic)
}

func proxyPythonScrape(c *gin.Context) {
	pythonURL := strings.TrimRight(os.Getenv("PY_BACKEND_URL"), "/")
	if pythonURL == "" {
		c.JSON(http.StatusNotImplemented, gin.H{"message": "Python scrape backend is not configured"})
		return
	}

	client := &http.Client{Timeout: 30 * time.Minute}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, pythonURL+"/scrape", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": err.Error()})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": err.Error()})
		return
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.DataFromReader(resp.StatusCode, int64(len(body)), contentType, bytes.NewReader(body), nil)
}

func queryInt(c *gin.Context, key string, fallback int) int {
	value, err := strconv.Atoi(c.DefaultQuery(key, strconv.Itoa(fallback)))
	if err != nil {
		return fallback
	}
	return value
}

func queryBool(c *gin.Context, key string) bool {
	return strings.EqualFold(c.DefaultQuery(key, "false"), "true")
}

func pathInt(c *gin.Context, key string) int {
	value, _ := strconv.Atoi(c.Param(key))
	return value
}

func applyComicPatch(comic *service.ComicJSON, patch map[string]any) {
	if titles, ok := stringSlice(patch["titles"]); ok {
		comic.Titles = titles
	}
	if value, ok := stringValue(patch["cover"]); ok {
		comic.Cover = value
	}
	if value, ok := stringValue(patch["description"]); ok {
		comic.Description = value
	}
	if value, ok := stringValue(patch["author"]); ok {
		comic.Author = value
	}
	if value, ok := intValue(patch["current_chap"]); ok {
		comic.CurrentChap = value
	}
	if value, ok := intValue(patch["viewed_chap"]); ok {
		comic.ViewedChap = value
	}
	if value, ok := intValue(patch["com_type"]); ok {
		comic.ComType = value
	}
	if value, ok := intValue(patch["status"]); ok {
		comic.Status = value
	}
	if value, ok := intValue(patch["rating"]); ok {
		comic.Rating = value
	}
	if value, ok := boolValue(patch["track"]); ok {
		comic.Track = value
	}
	if value, ok := boolValue(patch["cover_visible"]); ok {
		comic.CoverVisible = value
	}
	if values, ok := intSlice(patch["published_in"]); ok {
		comic.PublishedIn = values
	}
	if values, ok := intSlice(patch["genres"]); ok {
		comic.Genres = values
	}
}

func stringValue(value any) (string, bool) {
	parsed, ok := value.(string)
	return parsed, ok
}

func boolValue(value any) (bool, bool) {
	parsed, ok := value.(bool)
	return parsed, ok
}

func intValue(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	default:
		return 0, false
	}
}

func stringSlice(value any) ([]string, bool) {
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	values := make([]string, 0, len(items))
	for _, item := range items {
		if parsed, ok := item.(string); ok {
			values = append(values, parsed)
		}
	}
	return values, true
}

func intSlice(value any) ([]int, bool) {
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	values := make([]int, 0, len(items))
	for _, item := range items {
		if parsed, ok := intValue(item); ok {
			values = append(values, parsed)
		}
	}
	return values, true
}
