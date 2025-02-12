package sampler

import (
	"math/rand"
	"time"
)

const (
	charset             = "abcdefghijklmnopqrstuvwxyz" // 26 characters set
	defaultStrMinLength = 2
	defaultStrMaxLength = 16
)

var (
	weights = []float64{
		8.2, 1.5, 2.8, 4.3, 13.0, 2.2, 2.0, 6.1, 7.0, 0.2, 0.8, 4.0, 2.4,
		6.7, 7.5, 1.9, 0.1, 6.0, 6.3, 9.1, 2.8, 1.0, 2.4, 0.2, 2.0, 0.1,
	} // Approximate letter frequency in English
	cumulative       = make([]float64, len(weights))
	totalWeight      float64
	cumulativeWeight float64
)

func init() {
	for _, weight := range weights {
		totalWeight += weight
	}

	// Precompute cumulative probabilities
	// helps to determine where each character lies in the probability range.
	cumulativeWeight = 0.0
	for idx, weight := range weights {
		cumulativeWeight += weight
		cumulative[idx] = cumulativeWeight / totalWeight
	}
}

// randomString generates a random string of length between 2 and 16
func randomString() string {
	return randomStringOfLength(
		randomInt(defaultStrMinLength, defaultStrMaxLength))
}

// randomStringOfLength generates a random string of length provided
func randomStringOfLength(length int) string {
	// Generate random bytes slice for final string
	result := make([]byte, length)

	for rIdx := 0; rIdx < length; rIdx++ {
		randomValue := rand.Float64() // [0, 1)
		for charIdx, probability := range cumulative {
			if randomValue <= probability {
				result[rIdx] = charset[charIdx]
				break
			}
		}
	}
	return string(result)
}

// randomTimestamp generates a new proto timestamp between oneYearAgo and now
func randomTimestamp() time.Time {
	oneYearAgo := time.Now().Add(-time.Hour * 24 * 365).Unix()
	randomTime := rand.Int63n(time.Now().Unix()-oneYearAgo) + oneYearAgo
	randomNow := time.Unix(randomTime, 0)
	return randomNow
}

// randomTimestamp generates a timestamp since date provided and now
// if since is in the future, the timestamp will be the same as since
func randomTimestampSince(since time.Time) time.Time {
	if since.Unix() >= time.Now().Unix() {
		return since
	}
	randomTime := rand.Int63n(time.Now().Unix()-since.Unix()) + since.Unix()
	randomNow := time.Unix(randomTime, 0)
	return randomNow
}

// randomStringFromSet generates a random string from a set of strings
func randomStringFromSet(a ...string) string {
	if len(a) == 0 {
		return ""
	}
	return a[rand.Intn(len(a))]
}

// randomBool generates a random bool
func randomBool() bool {
	return rand.Intn(2) == 1
}

// randomInt generates a random int between min and max
func randomInt[T int | uint | int32 | uint32](min, max T) int {
	if min >= max {
		return 0
	}
	return rand.Intn(int(max-min)) + int(min)
}
