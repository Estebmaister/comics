package sampler

import (
	"math/rand"

	pb "comics/pkg/pb"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// NewComic generates a new comic with random values
func NewComic() *pb.Comic {
	currentChapter := uint32(randomUInt(0, 1000)) // #nosec G115
	comic := &pb.Comic{
		Id:          NewComicID(),
		Titles:      NewTitles(),
		Author:      NewAuthor(),
		Description: NewDescription(),
		ComType:     NewType(),
		Status:      NewStatus(),
		Cover:       NewCover(),
		CurrentChap: currentChapter,
		LastUpdate:  timestamppb.New(randomTimestamp()),
		PublishedIn: NewPublishers(),
		Genres:      NewGenres(),
		Rating:      NewRating(),
		Track:       randomBool(),
		ViewedChap:  uint32(randomUInt(0, currentChapter)), // #nosec G115
		Deleted:     randomBool(),
	}
	return comic
}

// NewComicID generates a random uint32 ID
func NewComicID() uint32 {
	return rand.Uint32() // #nosec G404
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
		return repeatedTitles[randomUInt(0, len(repeatedTitles))]
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
	setMaxLength := randomUInt(1, 5)

	repeatedPublishers := make([]pb.Publisher, 0)
	for i := 0; i < setMaxLength; i++ {
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
	setMaxLength := randomUInt(1, 5)
	repeatedGenres := make([]pb.Genre, 0)

	for i := 0; i < setMaxLength; i++ {
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

// NewType generates a random comic type
func NewType() pb.ComicType {
	return pb.ComicType(randomUInt(0, len(pb.ComicType_name))) // #nosec G115
}

// NewStatus generates a random status
func NewStatus() pb.Status {
	return pb.Status(randomUInt(0, len(pb.Status_name))) // #nosec G115
}

// NewRating generates a random rating
func NewRating() pb.Rating {
	return pb.Rating(randomUInt(0, len(pb.Rating_name))) // #nosec G115
}

func randomGenre() pb.Genre {
	return pb.Genre(randomUInt(0, len(pb.Genre_name))) // #nosec G115
}

func randomPublisher() pb.Publisher {
	return pb.Publisher(randomUInt(0, len(pb.Publisher_name))) // #nosec G115
}
