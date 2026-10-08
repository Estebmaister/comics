package route

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"comics/domain"
	"comics/internal/scrape"

	"github.com/gin-gonic/gin"
)

type comicJSON struct {
	ID           int      `json:"id"`
	Titles       []string `json:"titles"`
	CurrentChap  int      `json:"current_chap"`
	Cover        string   `json:"cover"`
	CoverVisible bool     `json:"cover_visible"`
	LastUpdate   string   `json:"last_update"`
	ComType      int      `json:"com_type"`
	Status       int      `json:"status"`
	PublishedIn  []int    `json:"published_in"`
	Genres       []int    `json:"genres"`
	Description  string   `json:"description"`
	Author       string   `json:"author"`
	Track        bool     `json:"track"`
	ViewedChap   int      `json:"viewed_chap"`
	Rating       int      `json:"rating"`
	Deleted      bool     `json:"deleted"`
}

func comicsRouter(comics domain.ComicUseCase, group *gin.RouterGroup) {
	group.GET("/health/db", func(c *gin.Context) {
		if err := comics.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "database unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	group.GET("/scrape", runScrape(comics))

	group.GET("/comics", listComics(comics))
	group.POST("/comics", createComic(comics))
	group.GET("/comics/:id", getComic(comics))
	group.PUT("/comics/:id", updateComic(comics))
	group.DELETE("/comics/:id", deleteComic(comics))
	group.PATCH("/comics/:id/cover-visibility", updateCoverVisibility(comics))
	group.GET("/comics/search/:title", searchComics(comics))
	group.PATCH("/comics/:id/:merging_id", mergeComics(comics))
	group.PUT("/comics/:id/:merging_id", mergeComics(comics))
}

func listComics(comics domain.ComicUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		from, ok := queryInt(c, "from", 0)
		if !ok {
			return
		}
		limit, ok := queryInt(c, "limit", 20)
		if !ok {
			return
		}
		ratingMin, ok := queryOptionalInt(c, "rating_min")
		if !ok {
			return
		}
		ratingMax, ok := queryOptionalInt(c, "rating_max")
		if !ok {
			return
		}
		result, err := comics.List(c.Request.Context(), domain.ComicListQuery{
			Offset:        from,
			Limit:         limit,
			OnlyTracked:   queryBool(c, "only_tracked"),
			OnlyUnchecked: queryBool(c, "only_unchecked"),
			Full:          queryBool(c, "full"),
			RatingMin:     ratingMin,
			RatingMax:     ratingMax,
			SortBy:        strings.TrimSpace(c.DefaultQuery("sort_by", "")),
			SortDir:       strings.TrimSpace(c.DefaultQuery("sort_dir", "")),
		})
		writeComicList(c, result, err)
	}
}

func searchComics(comics domain.ComicUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		title := strings.TrimSpace(c.Param("title"))
		if title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Title cannot be empty"})
			return
		}
		from, ok := queryInt(c, "from", 0)
		if !ok {
			return
		}
		limit, ok := queryInt(c, "limit", 20)
		if !ok {
			return
		}
		ratingMin, ok := queryOptionalInt(c, "rating_min")
		if !ok {
			return
		}
		ratingMax, ok := queryOptionalInt(c, "rating_max")
		if !ok {
			return
		}
		result, err := comics.Search(c.Request.Context(), domain.ComicListQuery{
			Offset:        from,
			Limit:         limit,
			SearchTitle:   title,
			OnlyTracked:   queryBool(c, "only_tracked"),
			OnlyUnchecked: queryBool(c, "only_unchecked"),
			Full:          queryBool(c, "full"),
			RatingMin:     ratingMin,
			RatingMax:     ratingMax,
			SortBy:        strings.TrimSpace(c.DefaultQuery("sort_by", "")),
			SortDir:       strings.TrimSpace(c.DefaultQuery("sort_dir", "")),
		})
		writeComicList(c, result, err)
	}
}

func getComic(comics domain.ComicUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathInt(c, "id")
		if !ok {
			return
		}
		comic, err := comics.Get(c.Request.Context(), id)
		writeComic(c, comic, err)
	}
}

func createComic(comics domain.ComicUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body map[string]any
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Body payload is necessary"})
			return
		}
		comic := domain.Comic{CoverVisible: true}
		applyComicPatch(&comic, body)
		if !hasNonEmptyTitle(comic.Titles) {
			c.JSON(http.StatusBadRequest, gin.H{"message": "titles should be a non-empty list of strings"})
			return
		}
		created, err := comics.Create(c.Request.Context(), comic)
		writeComicWithStatus(c, created, err, http.StatusCreated)
	}
}

func updateComic(comics domain.ComicUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body map[string]any
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Body payload is necessary"})
			return
		}
		id, ok := pathInt(c, "id")
		if !ok {
			return
		}
		patch := comicPatch(body)
		if patch.Titles != nil && !hasNonEmptyTitle(*patch.Titles) {
			c.JSON(http.StatusBadRequest, gin.H{"message": "titles should be a non-empty list of strings"})
			return
		}
		comic, err := comics.Update(c.Request.Context(), id, patch)
		writeComic(c, comic, err)
	}
}

func comicPatch(body map[string]any) domain.ComicPatch {
	patch := domain.ComicPatch{}
	if titles, ok := stringSlice(body["titles"]); ok {
		patch.Titles = &titles
	}
	if value, ok := stringValue(body["cover"]); ok {
		patch.Cover = &value
	}
	if value, ok := stringValue(body["description"]); ok {
		patch.Description = &value
	}
	if value, ok := stringValue(body["author"]); ok {
		patch.Author = &value
	}
	if value, ok := intValue(body["current_chap"]); ok {
		patch.CurrentChap = &value
	}
	if value, ok := intValue(body["viewed_chap"]); ok {
		patch.ViewedChap = &value
	}
	if value, ok := intValue(body["com_type"]); ok {
		patch.ComType = &value
	}
	if value, ok := intValue(body["status"]); ok {
		patch.Status = &value
	}
	if value, ok := intValue(body["rating"]); ok {
		patch.Rating = &value
	}
	if value, ok := boolValue(body["track"]); ok {
		patch.Track = &value
	}
	if value, ok := boolValue(body["deleted"]); ok {
		patch.Deleted = &value
	}
	if value, ok := boolValue(body["cover_visible"]); ok {
		patch.CoverVisible = &value
		patch.CoverVisiblePresent = true
	}
	if values, ok := intSlice(body["published_in"]); ok {
		patch.PublishedIn = &values
	}
	if values, ok := intSlice(body["genres"]); ok {
		patch.Genres = &values
	}
	return patch
}

func hasNonEmptyTitle(titles []string) bool {
	for _, title := range titles {
		if strings.TrimSpace(title) != "" {
			return true
		}
	}
	return false
}

func deleteComic(comics domain.ComicUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathInt(c, "id")
		if !ok {
			return
		}
		err := comics.Delete(c.Request.Context(), id)
		if errors.Is(err, domain.ErrComicNotFound) {
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

func updateCoverVisibility(comics domain.ComicUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Cover        string `json:"cover"`
			CoverVisible *bool  `json:"cover_visible"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Cover == "" || body.CoverVisible == nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "cover and cover_visible are required"})
			return
		}
		id, ok := pathInt(c, "id")
		if !ok {
			return
		}
		comic, err := comics.UpdateCoverVisibility(
			c.Request.Context(),
			id,
			body.Cover,
			*body.CoverVisible,
		)
		writeComic(c, comic, err)
	}
}

func mergeComics(comics domain.ComicUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		baseID, ok := pathInt(c, "id")
		if !ok {
			return
		}
		mergingID, ok := pathInt(c, "merging_id")
		if !ok {
			return
		}
		comic, err := comics.Merge(
			c.Request.Context(),
			baseID,
			mergingID,
		)
		if errors.Is(err, domain.ErrInvalidComicMerge) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		writeComic(c, comic, err)
	}
}

func writeComicList(c *gin.Context, result domain.ComicListResult, err error) {
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.Header("access-control-expose-headers", "total-comics,total-pages,current-page")
	c.Header("total-comics", strconv.Itoa(result.Total))
	c.Header("total-pages", strconv.Itoa(result.TotalPages))
	c.Header("current-page", strconv.Itoa(result.CurrentPage))
	c.JSON(http.StatusOK, comicJSONList(result.Comics))
}

func writeComic(c *gin.Context, comic domain.Comic, err error) {
	writeComicWithStatus(c, comic, err, http.StatusOK)
}

func writeComicWithStatus(c *gin.Context, comic domain.Comic, err error, status int) {
	if errors.Is(err, domain.ErrComicNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"message": "Comic not found"})
		return
	}
	if errors.Is(err, domain.ErrInvalidComicPayload) {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	if errors.Is(err, domain.ErrDuplicateComic) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Comic is already in the database"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(status, toComicJSON(comic))
}

func runScrape(comics domain.ComicUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Minute)
		defer cancel()
		if err := comics.Scrape(ctx); err != nil {
			if scrape.IsScrapeInProgress(err) {
				var busy *scrape.ErrScrapeInProgress
				if errors.As(err, &busy) {
					c.JSON(http.StatusConflict, gin.H{
						"message": "scrape already in progress",
						"source":  busy.Running,
					})
					return
				}
			}
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	}
}

func queryInt(c *gin.Context, key string, fallback int) (int, bool) {
	value, err := strconv.Atoi(c.DefaultQuery(key, strconv.Itoa(fallback)))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": key + " should be a valid integer"})
		return 0, false
	}
	return value, true
}

func queryBool(c *gin.Context, key string) bool {
	return strings.EqualFold(c.DefaultQuery(key, "false"), "true")
}

func queryOptionalInt(c *gin.Context, key string) (*int, bool) {
	raw, exists := c.GetQuery(key)
	if !exists || strings.TrimSpace(raw) == "" {
		return nil, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": key + " should be a valid integer"})
		return nil, false
	}
	return &value, true
}

func pathInt(c *gin.Context, key string) (int, bool) {
	value, err := strconv.Atoi(c.Param(key))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": key + " should be a valid integer"})
		return 0, false
	}
	return value, true
}

func applyComicPatch(comic *domain.Comic, patch map[string]any) {
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
	if value, ok := boolValue(patch["deleted"]); ok {
		comic.Deleted = value
	}
	if values, ok := intSlice(patch["published_in"]); ok {
		comic.PublishedIn = values
	}
	if values, ok := intSlice(patch["genres"]); ok {
		comic.Genres = values
	}
}

func toComicJSON(comic domain.Comic) comicJSON {
	return comicJSON{
		ID:           comic.ID,
		Titles:       comic.Titles,
		CurrentChap:  comic.CurrentChap,
		Cover:        comic.Cover,
		CoverVisible: comic.CoverVisible,
		LastUpdate:   formatComicTime(comic.LastUpdate),
		ComType:      comic.ComType,
		Status:       comic.Status,
		PublishedIn:  comic.PublishedIn,
		Genres:       comic.Genres,
		Description:  comic.Description,
		Author:       comic.Author,
		Track:        comic.Track,
		ViewedChap:   comic.ViewedChap,
		Rating:       comic.Rating,
		Deleted:      comic.Deleted,
	}
}

func comicJSONList(comics []domain.Comic) []comicJSON {
	items := make([]comicJSON, 0, len(comics))
	for _, comic := range comics {
		items = append(items, toComicJSON(comic))
	}
	return items
}

func formatComicTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
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
