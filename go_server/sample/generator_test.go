package sample

import (
	"comics/pb"
	"fmt"
	"reflect"
	"testing"
)

func TestNewComic(t *testing.T) {
	fakeComic := &pb.Comic{
		Id:     0,
		Titles: []string{"title"},
		Author: "author",
	}
	for i := 0; i < 100; i++ {
		newComic := NewComic()
		if reflect.DeepEqual(newComic, fakeComic) {
			t.Logf("newComic: %v", newComic)
			t.Error("NewComic() shouldn't be equal to fake")
		}
		if newComic.Titles[0] == "" {
			t.Logf("newComic: %v", newComic)
			t.Error("NewComic() should have a title")
		}
		if newComic.Cover == "" {
			t.Logf("newComic: %v", newComic)
			t.Error("NewComic() should have a cover")
		}
	}
}

func TestRandomFunctions(t *testing.T) {
	// Testing edge cases for randomInt
	zeroInt := randomInt(0, 0)
	zeroIntCheck1 := randomInt(9, 1)
	zeroIntCheck2 := randomInt(0, 1)
	if zeroIntCheck1 != 0 || zeroIntCheck2 != 0 || zeroInt != 0 {
		t.Error("randomInt( min >= max ) should return 0")
	}
	oneInt := randomInt(1, 2)
	if oneInt != 1 {
		t.Error("randomInt(1, 2) should return 1")
	}

	// Testing if randomInt generates a number between min and max
	for i := 0; i < 100; i++ {
		randomInt := randomInt(1, 10)
		if randomInt < 1 || randomInt > 10 {
			t.Errorf("randomInt(1, 10) = %d should be between 1 and 10", randomInt)
		}
	}

	// Testing strings from set edge cases
	str := randomStringFromSet("0")
	if str != "0" {
		t.Error("randomStringFromSet(\"0\") should return \"0\"")
	}
	emptyStr := randomStringFromSet()
	if emptyStr != "" {
		t.Errorf(
			"randomStringFromSet() = %#v should return an empty string",
			emptyStr)
	}

	// Testing that random flows on genres and publishers generators are executed
	for i := 0; i < 100; i++ {
		NewGenres()
		NewPublishers()
	}

	// Testing different genres and unknown genre are been generated
	var flagGenreUkn, flagGenreRandom bool
	for !flagGenreRandom || !flagGenreUkn {
		genres := NewGenres()
		fmt.Printf("%v\n", genres)
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
