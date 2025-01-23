package mysql

import (
	"context"
)

func (q *Queries) CreateComic(ctx context.Context, comic *Comic,
	titles []string, genres []int, publishers []int) error {

	err := q.InsertComic(ctx, InsertComicParams{
		Author:      comic.Author,
		Description: comic.Description,
		Cover:       comic.Cover,
		ComType:     comic.ComType,
		Status:      comic.Status,
		CurrentChap: comic.CurrentChap,
		ViewedChap:  comic.ViewedChap,
		LastUpdate:  comic.LastUpdate,
		Track:       comic.Track,
		Deleted:     comic.Deleted,
	})
	if err != nil {
		return err
	}

	comicID, err := q.GetLastInsertID(ctx)
	if err != nil {
		return err
	}

	for _, title := range titles {
		if err := q.InsertTitle(ctx, InsertTitleParams{
			ComicID: int32(comicID),
			Title:   title,
		}); err != nil {
			return err
		}
	}

	for _, genre := range genres {
		if err := q.InsertGenre(ctx, InsertGenreParams{
			ComicID: int32(comicID),
			Genre:   int32(genre),
		}); err != nil {
			return err
		}
	}

	for _, publisher := range publishers {
		if err := q.InsertPublisher(ctx, InsertPublisherParams{
			ComicID:   int32(comicID),
			Publisher: int32(publisher),
		}); err != nil {
			return err
		}
	}
	return nil
}
