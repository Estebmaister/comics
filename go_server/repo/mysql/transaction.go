package mysql

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"comics/domain"
	"comics/sql/mysql"
)

type ComicRepository struct {
	*Repo
}

func NewComicRepo(db *sql.DB) *ComicRepository {
	return &ComicRepository{
		Repo: NewRepo(db),
	}
}

// Comic model from DTO
func ComicFromDTO(
	comic *mysql.Comic,
	titles []*mysql.ComicTitle,
	genres []*mysql.ComicGenre,
	publishers []*mysql.ComicPublisher,
) *domain.Comic {
	return &domain.Comic{
		ID: int(comic.ID),
		Titles: func() []string {
			ts := make([]string, len(titles))
			for i, t := range titles {
				ts[i] = t.Title
			}
			return ts
		}(),
		Author:      comic.Author.String,
		Description: comic.Description.String,
		ComType:     int(comic.ComType),
		Status:      int(comic.Status),
		Cover:       comic.Cover.String,
		CurrentChap: int(comic.CurrentChap),
		LastUpdate:  comic.LastUpdate,
		Publishers: func() []int {
			ps := make([]int, len(publishers))
			for i, p := range publishers {
				ps[i] = int(p.Publisher)
			}
			return ps
		}(),
		Genres: func() []int {
			gs := make([]int, len(genres))
			for i, g := range genres {
				gs[i] = int(g.Genre)
			}
			return gs
		}(),
		Track:      comic.Track,
		ViewedChap: int(comic.ViewedChap),
		Deleted:    comic.Deleted,
	}
}

func ComicToDTO(comic *domain.Comic) *mysql.Comic {
	titles := make([]mysql.ComicTitle, len(comic.Titles))
	for i, t := range comic.Titles {
		titles[i] = mysql.ComicTitle{ComicID: int32(comic.ID), Title: t}
	}
	publishers := make([]mysql.ComicPublisher, len(comic.Publishers))
	for i, p := range comic.Publishers {
		publishers[i] = mysql.ComicPublisher{ComicID: int32(comic.ID), Publisher: int32(p)}
	}
	genres := make([]mysql.ComicGenre, len(comic.Genres))
	for i, g := range comic.Genres {
		genres[i] = mysql.ComicGenre{ComicID: int32(comic.ID), Genre: int32(g)}
	}
	return &mysql.Comic{
		ID:          int32(comic.ID),
		Author:      sql.NullString{String: comic.Author, Valid: true},
		Description: sql.NullString{String: comic.Description, Valid: true},
		Cover:       sql.NullString{String: comic.Cover, Valid: true},
		ComType:     int32(comic.ComType),
		Status:      int32(comic.Status),
		CurrentChap: int32(comic.CurrentChap),
		LastUpdate:  comic.LastUpdate,
		Track:       comic.Track,
		ViewedChap:  int32(comic.ViewedChap),
		Deleted:     comic.Deleted,
	}
}

func (r *ComicRepository) Create(ctx context.Context, comic *domain.Comic) error {
	return r.WithTx(ctx, func(q *mysql.Queries) error {
		comicDTO := ComicToDTO(comic)
		// Use the transactional queries to create the comic
		return q.CreateComic(ctx, comicDTO, comic.Titles, comic.Genres, comic.Publishers)
	})
}

func (r *ComicRepository) GetByID(ctx context.Context, id int) (*domain.Comic, error) {
	comic, err := r.queries.GetComicById(ctx, int32(id))
	if err != nil {
		return nil, err
	}
	comicTitles := strings.Split(comic.Titles.String, "|")
	titles := make([]string, len(comicTitles))
	for i, t := range comicTitles {
		titles[i] = t
	}
	comicGenres := strings.Split(comic.Genres.String, ",")
	genres := make([]int, len(comicGenres))
	for i, g := range comicGenres {
		gsi, _ := strconv.Atoi(g)
		genres[i] = gsi
	}
	comicPublishers := strings.Split(comic.PublishedIn.String, ",")
	publishers := make([]int, len(comicPublishers))
	for i, p := range comicPublishers {
		psi, _ := strconv.Atoi(p)
		publishers[i] = psi
	}
	comicModel := &domain.Comic{
		ID:          int(comic.ID),
		Titles:      titles,
		Author:      comic.Author.String,
		Description: comic.Description.String,
		ComType:     int(comic.ComType),
		Status:      int(comic.Status),
		Cover:       comic.Cover.String,
		CurrentChap: int(comic.CurrentChap),
		LastUpdate:  comic.LastUpdate,
		Publishers:  publishers,
		Genres:      genres,
		Track:       comic.Track,
		ViewedChap:  int(comic.ViewedChap),
		Deleted:     comic.Deleted,
	}
	return comicModel, err
}
