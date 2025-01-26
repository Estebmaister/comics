package sampler

import (
	"reflect"
	"testing"

	pb "comics/pkg/pb"
)

const (
	newComicPlaceHolder = "newComic: %v"
)

func TestNewComic(t *testing.T) {
	t.Parallel()
	fakeComic := &pb.Comic{
		Id:     0,
		Titles: []string{"title"},
		Author: "author",
	}
	for i := 0; i < 100; i++ {
		newComic := NewComic()
		if reflect.DeepEqual(newComic, fakeComic) {
			t.Logf(newComicPlaceHolder, newComic)
			t.Error("NewComic() shouldn't be equal to fake")
		}
		if newComic.Titles[0] == "" {
			t.Logf(newComicPlaceHolder, newComic)
			t.Error("NewComic() should have a title")
		}
		if newComic.Cover == "" {
			t.Logf(newComicPlaceHolder, newComic)
			t.Error("NewComic() should have a cover")
		}
	}

}

func TestNewComicAuxFunctions(t *testing.T) {
	t.Parallel()
	// Testing that random flows on genres and publishers generators are executed
	for i := 0; i < 100; i++ {
		NewGenres()
		NewPublishers()
	}
	// Testing different genres and unknown genre are been generated
	var flagGenreUkn, flagGenreRandom bool
	for !flagGenreRandom || !flagGenreUkn {
		genres := NewGenres()
		if genres[0] == pb.Genre_GENRE_UNKNOWN {
			flagGenreUkn = true
		} else {
			flagGenreRandom = true
		}
	}

	// Testing single titles and multi titles are been generated
	var flagSingleTitle, flagMultiTitles bool
	for !flagSingleTitle || !flagMultiTitles {
		titles := NewTitles()
		println(titles)
		if len(titles) == 1 {
			flagSingleTitle = true
		} else {
			flagMultiTitles = true
		}
	}
}
