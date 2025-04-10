package sampler

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestRandomInt(t *testing.T) {
	t.Parallel()
	// Testing edge cases for randomUInt
	zeroInt := randomUInt(0, 0)
	zeroIntCheck1 := randomUInt(9, 1)
	zeroIntCheck2 := randomUInt(0, 1)
	if zeroIntCheck1 != 0 || zeroIntCheck2 != 0 || zeroInt != 0 {
		t.Error("randomUInt( min >= max ) should return 0")
	}
	oneInt := randomUInt(1, 2)
	if oneInt != 1 {
		t.Error("randomUInt(1, 2) should return 1")
	}

	// Testing if randomUInt generates a number between min and max
	for i := 0; i < 100; i++ {
		randomUInt := randomUInt(1, 10)
		if randomUInt < 1 || randomUInt > 10 {
			t.Errorf("randomUInt(1, 10) = %d should be between 1 and 10", randomUInt)
		}
	}
}

func TestRandomStrings(t *testing.T) {
	t.Parallel()
	// Testing randomStringFromSet edge cases
	emptyStr := randomStringFromSet()
	if emptyStr != "" {
		t.Errorf(
			"randomStringFromSet() = %#v should return an empty string",
			emptyStr)
	}
	oneString := randomStringFromSet("a")
	if oneString != "a" {
		t.Errorf("randomStringFromSet(\"a\") should return \"a\"")
	}

	// Testing randomString
	for i := 0; i < 100; i++ {
		randomString := RandomString()
		if len(randomString) < defaultStrMinLength || len(randomString) > defaultStrMaxLength {
			t.Errorf("randomString() = %s should be between 2 and 16", randomString)
		}
	}
}

func TestRandomTimestamps(t *testing.T) {
	t.Parallel()
	// Testing if randomTimestamp generates a timestamp between min and max
	minTimestamp := time.Now().Add(-time.Hour * 24 * 365)
	maxTimestamp := time.Now()
	for i := 0; i < 100; i++ {
		randomTimestamp := randomTimestamp()
		if randomTimestamp.Before(minTimestamp) || randomTimestamp.After(maxTimestamp) {
			t.Errorf("randomTimestamp() = %v should be between %v and %v",
				randomTimestamp, minTimestamp, maxTimestamp)
		}
	}

	// Testing edge cases for randomTimestampSince

	// "since" is in the future, the timestamp should be the same as since
	timeFuture := time.Now().Add(time.Minute)
	nowTimestamp := randomTimestampSince(timeFuture)
	if nowTimestamp.Compare(timeFuture) != 0 {
		diff := cmp.Diff(timeFuture, nowTimestamp)
		t.Errorf("randomTimestampSince('future') difference %v", diff)
	}

	// "since" is in the past, the timestamp should be between since and now
	timePast := time.Now().Add(-time.Minute)
	middleTimestamp := randomTimestampSince(timePast)
	if middleTimestamp.Before(timePast) || middleTimestamp.After(time.Now()) {
		t.Errorf("(%v) should be between %v and %v", middleTimestamp, timePast, timeFuture)
	}
}
