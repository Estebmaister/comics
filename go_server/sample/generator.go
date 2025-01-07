package sample

import (
	"comics/pb"
	"math/rand"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// NewComic generates a new comic with random values
func NewComic() *pb.Comic {
	currentChapter := uint32(randomInt(0, 1000))
	comic := &pb.Comic{
		Id:          NewID(),
		Titles:      NewTitles(),
		Author:      NewAuthor(),
		Description: NewDescription(),
		Type:        NewType(),
		Status:      NewStatus(),
		Cover:       NewCover(),
		CurrentChap: currentChapter,
		LastUpdate:  NewLastUpdate(),
		Publishers:  NewPublishers(),
		Genres:      NewGenres(),
		Rating:      NewRating(),
		Track:       randomBool(),
		ViewedChap:  uint32(randomInt(0, currentChapter)),
		Deleted:     randomBool(),
	}
	return comic
}

// NewID generates a random ID
func NewID() uint32 {
	return rand.Uint32()
}

// NewTitles generates a random title or a set of repeated titles
func NewTitles() []string {
	titles := []string{
		"Dragon Ball Z", "One Punch Man", "Sword Art Online",
		"The Promised Neverland", "Berserk", "Fairy Tail", "Haikyuu",
		"Solo Leveling", "God of High School", "Boku no Hero Academia",
		"Pandora", "Dragon Ball", "Bleach", "My Hero Academia", "Death Note",
		"Black Clover", "Jujutsu Kaisen", "Chainsaw Man", "Kuroko no Basket",
	}
	repeatedTitles := [][]string{
		{"Naruto", "Boruto", "Uzumaki", "Naruto Shippuden"},
		{"One Piece", "The one piece", "The Pirate King"},
		{"Attack on Titan", "Shingeki no Kyojin", "AOT"},
		{"Fullmetal Alchemist", "Fullmetal Alchemist Brotherhood"},
		{"Hunter x Hunter", "Hunter x Hunter 2011"},
		{"Tokyo Ghoul", "Tokyo Ghoul:re"},
		{"Nanatsu no Taizai", "The Seven Deadly Sins"},
		{"Tokyo Revengers", "Tokyo Revengers 2017"},
		{"God's tower", "Tower of God"},
		{"Kimetsu no yaiba", "Demon Slayer"},
	}
	if randomBool() {
		return repeatedTitles[randomInt(0, len(repeatedTitles))]
	}
	return []string{randomStringFromSet(titles...)}
}

// NewAuthor generates a random author
func NewAuthor() string {
	author := []string{"", "Stan Lee", "Masashi Kishimoto", "Eiichiro Oda",
		"Akira Toriyama", "Hajime Isayama", "Kohei Horikoshi", "Yoshihiro Togashi"}
	return randomStringFromSet(author...)
}

// NewDescription generates a random description
func NewDescription() string {
	description := []string{
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit, " +
			"sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.",
		"Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi " +
			"ut aliquip ex ea commodo consequat.",
		"Duis aute irure dolor in reprehenderit in voluptate velit esse cillum " +
			"dolore eu fugiat nulla pariatur.",
		"Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia " +
			"deserunt mollit anim id est laborum.",
		"",
	}
	return randomStringFromSet(description...)
}

// NewCover generates a random cover URL string
func NewCover() string {
	// list of possible cover URLs
	covers := []string{
		"https://www.google.com",
		"https://www.bing.com",
		"https://www.yahoo.com",
	}
	// return a random cover URL
	return randomStringFromSet(covers...)
}

// NewPublishers generates a random set of publishers (1 to 5)
func NewPublishers() []pb.Publisher {
	seen := make(map[int]bool) // keep track of seen publishers

	repeatedPublishers := make([]pb.Publisher, randomInt(1, 5))
	for i := 0; i < len(repeatedPublishers); i++ {
		newPublisher := randomPublisher()
		// if newPublisher in repeatedPublishers continue
		if _, ok := seen[int(newPublisher)]; ok {
			continue
		}
		repeatedPublishers = append(repeatedPublishers, newPublisher)
		seen[int(newPublisher)] = true
	}

	return repeatedPublishers
}

// NewGenres generates a random genre or a set of repeated genres (1 to 5)
func NewGenres() []pb.Genre {
	seen := make(map[int]bool) // keep track of seen genres

	repeatedGenres := make([]pb.Genre, randomInt(1, 5))
	for i := 0; i < len(repeatedGenres); i++ {
		newGenre := randomGenre()
		// if newGenre in repeatedGenres continue
		if _, ok := seen[int(newGenre)]; ok {
			continue
		}
		repeatedGenres = append(repeatedGenres, newGenre)
		seen[int(newGenre)] = true
	}

	if randomBool() {
		return repeatedGenres
	} // 50% chance of returning an unknown genre
	return []pb.Genre{pb.Genre_GENRE_UNKNOWN}
}

// NewLatUpdate
func NewLastUpdate() *timestamppb.Timestamp {
	// generate a random time between 1st Jan 2000 and now
	randomTime := rand.Int63n(time.Now().Unix()-94608000) + 94608000
	// convert randomTime to time.Time
	randomNow := time.Unix(randomTime, 0)

	return timestamppb.New(randomNow)
}

// NewType generates a random comic type
func NewType() pb.ComicType {
	return pb.ComicType(randomInt(0, 5))
}

// NewStatus generates a random status
func NewStatus() pb.Status {
	return pb.Status(randomInt(0, 5))
}

// NewRating generates a random rating
func NewRating() pb.Rating {
	return pb.Rating(randomInt(0, 10))
}

func randomStringFromSet(a ...string) string {
	if len(a) == 0 {
		return ""
	}
	return a[rand.Intn(len(a))]
}

func randomBool() bool {
	return rand.Intn(2) == 1
}

func randomInt[T int | uint | int32 | uint32](min, max T) int {
	if min >= max {
		return 0
	}
	return rand.Intn(int(max-min)) + int(min)
}

func randomGenre() pb.Genre {
	return pb.Genre(randomInt(0, len(pb.Genre_name)))
}

func randomPublisher() pb.Publisher {
	return pb.Publisher(randomInt(0, len(pb.Publisher_name)))
}
