// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.35.2
// 	protoc        v5.29.1
// source: comics.proto

package pb

import (
	_ "github.com/envoyproxy/protoc-gen-validate/validate"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Comic types enum
// Represents different types of comics available in the system
type ComicType int32

const (
	ComicType_TYPE_UNKNOWN ComicType = 0
	ComicType_MANGA        ComicType = 1
	ComicType_MANHUA       ComicType = 2
	ComicType_MANHWA       ComicType = 3
	ComicType_WEBTOON      ComicType = 3 // Alias for MANHWA
	ComicType_NOVEL        ComicType = 4
)

// Enum value maps for ComicType.
var (
	ComicType_name = map[int32]string{
		0: "TYPE_UNKNOWN",
		1: "MANGA",
		2: "MANHUA",
		3: "MANHWA",
		// Duplicate value: 3: "WEBTOON",
		4: "NOVEL",
	}
	ComicType_value = map[string]int32{
		"TYPE_UNKNOWN": 0,
		"MANGA":        1,
		"MANHUA":       2,
		"MANHWA":       3,
		"WEBTOON":      3,
		"NOVEL":        4,
	}
)

func (x ComicType) Enum() *ComicType {
	p := new(ComicType)
	*p = x
	return p
}

func (x ComicType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ComicType) Descriptor() protoreflect.EnumDescriptor {
	return file_comics_proto_enumTypes[0].Descriptor()
}

func (ComicType) Type() protoreflect.EnumType {
	return &file_comics_proto_enumTypes[0]
}

func (x ComicType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ComicType.Descriptor instead.
func (ComicType) EnumDescriptor() ([]byte, []int) {
	return file_comics_proto_rawDescGZIP(), []int{0}
}

// Comic status enum
// Represents the current publication status of a comic
type Status int32

const (
	Status_STATUS_UNKNOWN Status = 0
	Status_COMPLETED      Status = 1
	Status_ON_AIR         Status = 2
	Status_BREAK          Status = 3
	Status_DROPPED        Status = 4
)

// Enum value maps for Status.
var (
	Status_name = map[int32]string{
		0: "STATUS_UNKNOWN",
		1: "COMPLETED",
		2: "ON_AIR",
		3: "BREAK",
		4: "DROPPED",
	}
	Status_value = map[string]int32{
		"STATUS_UNKNOWN": 0,
		"COMPLETED":      1,
		"ON_AIR":         2,
		"BREAK":          3,
		"DROPPED":        4,
	}
)

func (x Status) Enum() *Status {
	p := new(Status)
	*p = x
	return p
}

func (x Status) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Status) Descriptor() protoreflect.EnumDescriptor {
	return file_comics_proto_enumTypes[1].Descriptor()
}

func (Status) Type() protoreflect.EnumType {
	return &file_comics_proto_enumTypes[1]
}

func (x Status) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Status.Descriptor instead.
func (Status) EnumDescriptor() ([]byte, []int) {
	return file_comics_proto_rawDescGZIP(), []int{1}
}

// Comic genres enum
// Represents different genres that can be assigned to a comic
type Genre int32

const (
	Genre_GENRE_UNKNOWN Genre = 0
	Genre_ACTION        Genre = 1
	Genre_ADVENTURE     Genre = 2
	Genre_FANTASY       Genre = 3
	Genre_OVERPOWERED   Genre = 4
	Genre_COMEDY        Genre = 5
	Genre_DRAMA         Genre = 6
	Genre_SCHOOL_LIFE   Genre = 7
	Genre_SYSTEM        Genre = 8
	Genre_SUPERNATURAL  Genre = 9
	Genre_MARTIAL_ARTS  Genre = 10
	Genre_ROMANCE       Genre = 11
	Genre_SHOUNEN       Genre = 12
	Genre_REINCARNATION Genre = 13
	// Common aliases
	Genre_OP          Genre = 4  // Alias for OVERPOWERED
	Genre_CULTIVATION Genre = 10 // Alias for MARTIAL_ARTS
)

// Enum value maps for Genre.
var (
	Genre_name = map[int32]string{
		0:  "GENRE_UNKNOWN",
		1:  "ACTION",
		2:  "ADVENTURE",
		3:  "FANTASY",
		4:  "OVERPOWERED",
		5:  "COMEDY",
		6:  "DRAMA",
		7:  "SCHOOL_LIFE",
		8:  "SYSTEM",
		9:  "SUPERNATURAL",
		10: "MARTIAL_ARTS",
		11: "ROMANCE",
		12: "SHOUNEN",
		13: "REINCARNATION",
		// Duplicate value: 4: "OP",
		// Duplicate value: 10: "CULTIVATION",
	}
	Genre_value = map[string]int32{
		"GENRE_UNKNOWN": 0,
		"ACTION":        1,
		"ADVENTURE":     2,
		"FANTASY":       3,
		"OVERPOWERED":   4,
		"COMEDY":        5,
		"DRAMA":         6,
		"SCHOOL_LIFE":   7,
		"SYSTEM":        8,
		"SUPERNATURAL":  9,
		"MARTIAL_ARTS":  10,
		"ROMANCE":       11,
		"SHOUNEN":       12,
		"REINCARNATION": 13,
		"OP":            4,
		"CULTIVATION":   10,
	}
)

func (x Genre) Enum() *Genre {
	p := new(Genre)
	*p = x
	return p
}

func (x Genre) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Genre) Descriptor() protoreflect.EnumDescriptor {
	return file_comics_proto_enumTypes[2].Descriptor()
}

func (Genre) Type() protoreflect.EnumType {
	return &file_comics_proto_enumTypes[2]
}

func (x Genre) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Genre.Descriptor instead.
func (Genre) EnumDescriptor() ([]byte, []int) {
	return file_comics_proto_rawDescGZIP(), []int{2}
}

// Publishers enum
// Represents different comic publishers/scan groups
type Publisher int32

const (
	Publisher_PUBLISHER_UNKNOWN Publisher = 0
	Publisher_ASURA             Publisher = 1
	Publisher_REAPER_SCANS      Publisher = 2
	Publisher_MANHUA_PLUS       Publisher = 3
	Publisher_FLAME_SCANS       Publisher = 4
	Publisher_LUMINOUS_SCANS    Publisher = 5
	Publisher_RESET_SCANS       Publisher = 6
	Publisher_ISEKAI_SCAN       Publisher = 7
	Publisher_REALM_SCANS       Publisher = 8
	Publisher_LEVIATAN_SCANS    Publisher = 9
	Publisher_NIGHT_SCANS       Publisher = 10
	Publisher_VOID_SCANS        Publisher = 11
	Publisher_DRAKE_SCANS       Publisher = 12
	Publisher_NOVEL_MIC         Publisher = 13
)

// Enum value maps for Publisher.
var (
	Publisher_name = map[int32]string{
		0:  "PUBLISHER_UNKNOWN",
		1:  "ASURA",
		2:  "REAPER_SCANS",
		3:  "MANHUA_PLUS",
		4:  "FLAME_SCANS",
		5:  "LUMINOUS_SCANS",
		6:  "RESET_SCANS",
		7:  "ISEKAI_SCAN",
		8:  "REALM_SCANS",
		9:  "LEVIATAN_SCANS",
		10: "NIGHT_SCANS",
		11: "VOID_SCANS",
		12: "DRAKE_SCANS",
		13: "NOVEL_MIC",
	}
	Publisher_value = map[string]int32{
		"PUBLISHER_UNKNOWN": 0,
		"ASURA":             1,
		"REAPER_SCANS":      2,
		"MANHUA_PLUS":       3,
		"FLAME_SCANS":       4,
		"LUMINOUS_SCANS":    5,
		"RESET_SCANS":       6,
		"ISEKAI_SCAN":       7,
		"REALM_SCANS":       8,
		"LEVIATAN_SCANS":    9,
		"NIGHT_SCANS":       10,
		"VOID_SCANS":        11,
		"DRAKE_SCANS":       12,
		"NOVEL_MIC":         13,
	}
)

func (x Publisher) Enum() *Publisher {
	p := new(Publisher)
	*p = x
	return p
}

func (x Publisher) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Publisher) Descriptor() protoreflect.EnumDescriptor {
	return file_comics_proto_enumTypes[3].Descriptor()
}

func (Publisher) Type() protoreflect.EnumType {
	return &file_comics_proto_enumTypes[3]
}

func (x Publisher) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Publisher.Descriptor instead.
func (Publisher) EnumDescriptor() ([]byte, []int) {
	return file_comics_proto_rawDescGZIP(), []int{3}
}

// Rating enum
// Represents user ratings for comics
type Rating int32

const (
	Rating_RATING_UNKNOWN Rating = 0
	Rating_F_RATED        Rating = 1
	Rating_E_RATED        Rating = 2
	Rating_D_RATED        Rating = 3
	Rating_C_RATED        Rating = 4
	Rating_B_RATED        Rating = 5
	Rating_A_RATED        Rating = 6
	Rating_S_RATED        Rating = 7
	Rating_SS_RATED       Rating = 8
	Rating_SSS_RATED      Rating = 9
	// Common aliases
	Rating_F           Rating = 1 // Alias for F_RATED
	Rating_ONE_STAR    Rating = 1 // Alias for F_RATED
	Rating_E           Rating = 2 // Alias for E_RATED
	Rating_D           Rating = 3 // Alias for D_RATED
	Rating_TWO_STARS   Rating = 3 // Alias for D_RATED
	Rating_C           Rating = 4 // Alias for C_RATED
	Rating_B           Rating = 5 // Alias for B_RATED
	Rating_THREE_STARS Rating = 5 // Alias for B_RATED
	Rating_A           Rating = 6 // Alias for A_RATED
	Rating_S           Rating = 7 // Alias for S_RATED
	Rating_FOUR_STARS  Rating = 7 // Alias for S_RATED
	Rating_SS          Rating = 8 // Alias for SS_RATED
	Rating_SSS         Rating = 9 // Alias for SSS_RATED
	Rating_FIVE_STARS  Rating = 9 // Alias for SSS_RATED
)

// Enum value maps for Rating.
var (
	Rating_name = map[int32]string{
		0: "RATING_UNKNOWN",
		1: "F_RATED",
		2: "E_RATED",
		3: "D_RATED",
		4: "C_RATED",
		5: "B_RATED",
		6: "A_RATED",
		7: "S_RATED",
		8: "SS_RATED",
		9: "SSS_RATED",
		// Duplicate value: 1: "F",
		// Duplicate value: 1: "ONE_STAR",
		// Duplicate value: 2: "E",
		// Duplicate value: 3: "D",
		// Duplicate value: 3: "TWO_STARS",
		// Duplicate value: 4: "C",
		// Duplicate value: 5: "B",
		// Duplicate value: 5: "THREE_STARS",
		// Duplicate value: 6: "A",
		// Duplicate value: 7: "S",
		// Duplicate value: 7: "FOUR_STARS",
		// Duplicate value: 8: "SS",
		// Duplicate value: 9: "SSS",
		// Duplicate value: 9: "FIVE_STARS",
	}
	Rating_value = map[string]int32{
		"RATING_UNKNOWN": 0,
		"F_RATED":        1,
		"E_RATED":        2,
		"D_RATED":        3,
		"C_RATED":        4,
		"B_RATED":        5,
		"A_RATED":        6,
		"S_RATED":        7,
		"SS_RATED":       8,
		"SSS_RATED":      9,
		"F":              1,
		"ONE_STAR":       1,
		"E":              2,
		"D":              3,
		"TWO_STARS":      3,
		"C":              4,
		"B":              5,
		"THREE_STARS":    5,
		"A":              6,
		"S":              7,
		"FOUR_STARS":     7,
		"SS":             8,
		"SSS":            9,
		"FIVE_STARS":     9,
	}
)

func (x Rating) Enum() *Rating {
	p := new(Rating)
	*p = x
	return p
}

func (x Rating) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Rating) Descriptor() protoreflect.EnumDescriptor {
	return file_comics_proto_enumTypes[4].Descriptor()
}

func (Rating) Type() protoreflect.EnumType {
	return &file_comics_proto_enumTypes[4]
}

func (x Rating) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Rating.Descriptor instead.
func (Rating) EnumDescriptor() ([]byte, []int) {
	return file_comics_proto_rawDescGZIP(), []int{4}
}

// Comics collection message
type Comics struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Comics []*Comic `protobuf:"bytes,1,rep,name=comics,proto3" json:"comics,omitempty"`
}

func (x *Comics) Reset() {
	*x = Comics{}
	mi := &file_comics_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Comics) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Comics) ProtoMessage() {}

func (x *Comics) ProtoReflect() protoreflect.Message {
	mi := &file_comics_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Comics.ProtoReflect.Descriptor instead.
func (*Comics) Descriptor() ([]byte, []int) {
	return file_comics_proto_rawDescGZIP(), []int{0}
}

func (x *Comics) GetComics() []*Comic {
	if x != nil {
		return x.Comics
	}
	return nil
}

// Comic message definition
// Represents a comic book with all its metadata and user interaction
// information
type Comic struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Primary identifiers
	Id     uint32   `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Titles []string `protobuf:"bytes,2,rep,name=titles,proto3" json:"titles,omitempty"` // At least one title required
	// Basic information
	Author      string    `protobuf:"bytes,3,opt,name=author,proto3" json:"author,omitempty"`
	Description string    `protobuf:"bytes,4,opt,name=description,proto3" json:"description,omitempty"`
	Type        ComicType `protobuf:"varint,5,opt,name=type,proto3,enum=comics.ComicType" json:"type,omitempty"`
	Status      Status    `protobuf:"varint,6,opt,name=status,proto3,enum=comics.Status" json:"status,omitempty"`
	// Content metadata
	Cover       string                 `protobuf:"bytes,7,opt,name=cover,proto3" json:"cover,omitempty"` // Must be a valid URI
	CurrentChap uint32                 `protobuf:"varint,8,opt,name=current_chap,json=currentChap,proto3" json:"current_chap,omitempty"`
	LastUpdate  *timestamppb.Timestamp `protobuf:"bytes,9,opt,name=last_update,json=lastUpdate,proto3" json:"last_update,omitempty"` // Using standard timestamp
	// Classifications
	Publishers []Publisher `protobuf:"varint,10,rep,packed,name=publishers,proto3,enum=comics.Publisher" json:"publishers,omitempty"`
	Genres     []Genre     `protobuf:"varint,11,rep,packed,name=genres,proto3,enum=comics.Genre" json:"genres,omitempty"`
	Rating     Rating      `protobuf:"varint,12,opt,name=rating,proto3,enum=comics.Rating" json:"rating,omitempty"`
	// User interaction fields
	Track      bool   `protobuf:"varint,13,opt,name=track,proto3" json:"track,omitempty"`
	ViewedChap uint32 `protobuf:"varint,14,opt,name=viewed_chap,json=viewedChap,proto3" json:"viewed_chap,omitempty"`
	Deleted    bool   `protobuf:"varint,15,opt,name=deleted,proto3" json:"deleted,omitempty"`
}

func (x *Comic) Reset() {
	*x = Comic{}
	mi := &file_comics_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Comic) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Comic) ProtoMessage() {}

func (x *Comic) ProtoReflect() protoreflect.Message {
	mi := &file_comics_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Comic.ProtoReflect.Descriptor instead.
func (*Comic) Descriptor() ([]byte, []int) {
	return file_comics_proto_rawDescGZIP(), []int{1}
}

func (x *Comic) GetId() uint32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Comic) GetTitles() []string {
	if x != nil {
		return x.Titles
	}
	return nil
}

func (x *Comic) GetAuthor() string {
	if x != nil {
		return x.Author
	}
	return ""
}

func (x *Comic) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *Comic) GetType() ComicType {
	if x != nil {
		return x.Type
	}
	return ComicType_TYPE_UNKNOWN
}

func (x *Comic) GetStatus() Status {
	if x != nil {
		return x.Status
	}
	return Status_STATUS_UNKNOWN
}

func (x *Comic) GetCover() string {
	if x != nil {
		return x.Cover
	}
	return ""
}

func (x *Comic) GetCurrentChap() uint32 {
	if x != nil {
		return x.CurrentChap
	}
	return 0
}

func (x *Comic) GetLastUpdate() *timestamppb.Timestamp {
	if x != nil {
		return x.LastUpdate
	}
	return nil
}

func (x *Comic) GetPublishers() []Publisher {
	if x != nil {
		return x.Publishers
	}
	return nil
}

func (x *Comic) GetGenres() []Genre {
	if x != nil {
		return x.Genres
	}
	return nil
}

func (x *Comic) GetRating() Rating {
	if x != nil {
		return x.Rating
	}
	return Rating_RATING_UNKNOWN
}

func (x *Comic) GetTrack() bool {
	if x != nil {
		return x.Track
	}
	return false
}

func (x *Comic) GetViewedChap() uint32 {
	if x != nil {
		return x.ViewedChap
	}
	return 0
}

func (x *Comic) GetDeleted() bool {
	if x != nil {
		return x.Deleted
	}
	return false
}

var File_comics_proto protoreflect.FileDescriptor

var file_comics_proto_rawDesc = []byte{
	0x0a, 0x0c, 0x63, 0x6f, 0x6d, 0x69, 0x63, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x06,
	0x63, 0x6f, 0x6d, 0x69, 0x63, 0x73, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x17, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74,
	0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x22, 0x2f, 0x0a, 0x06, 0x43, 0x6f, 0x6d, 0x69, 0x63, 0x73, 0x12, 0x25, 0x0a, 0x06, 0x63, 0x6f,
	0x6d, 0x69, 0x63, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x63, 0x6f, 0x6d,
	0x69, 0x63, 0x73, 0x2e, 0x43, 0x6f, 0x6d, 0x69, 0x63, 0x52, 0x06, 0x63, 0x6f, 0x6d, 0x69, 0x63,
	0x73, 0x22, 0xa8, 0x04, 0x0a, 0x05, 0x43, 0x6f, 0x6d, 0x69, 0x63, 0x12, 0x0e, 0x0a, 0x02, 0x69,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x02, 0x69, 0x64, 0x12, 0x26, 0x0a, 0x06, 0x74,
	0x69, 0x74, 0x6c, 0x65, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x42, 0x0e, 0xfa, 0x42, 0x0b,
	0x92, 0x01, 0x08, 0x08, 0x01, 0x22, 0x04, 0x72, 0x02, 0x10, 0x01, 0x52, 0x06, 0x74, 0x69, 0x74,
	0x6c, 0x65, 0x73, 0x12, 0x16, 0x0a, 0x06, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x06, 0x61, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x12, 0x20, 0x0a, 0x0b, 0x64,
	0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x25, 0x0a,
	0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x11, 0x2e, 0x63, 0x6f,
	0x6d, 0x69, 0x63, 0x73, 0x2e, 0x43, 0x6f, 0x6d, 0x69, 0x63, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04,
	0x74, 0x79, 0x70, 0x65, 0x12, 0x26, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x06,
	0x20, 0x01, 0x28, 0x0e, 0x32, 0x0e, 0x2e, 0x63, 0x6f, 0x6d, 0x69, 0x63, 0x73, 0x2e, 0x53, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x1e, 0x0a, 0x05,
	0x63, 0x6f, 0x76, 0x65, 0x72, 0x18, 0x07, 0x20, 0x01, 0x28, 0x09, 0x42, 0x08, 0xfa, 0x42, 0x05,
	0x72, 0x03, 0x88, 0x01, 0x01, 0x52, 0x05, 0x63, 0x6f, 0x76, 0x65, 0x72, 0x12, 0x21, 0x0a, 0x0c,
	0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x5f, 0x63, 0x68, 0x61, 0x70, 0x18, 0x08, 0x20, 0x01,
	0x28, 0x0d, 0x52, 0x0b, 0x63, 0x75, 0x72, 0x72, 0x65, 0x6e, 0x74, 0x43, 0x68, 0x61, 0x70, 0x12,
	0x3b, 0x0a, 0x0b, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x18, 0x09,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x52, 0x0a, 0x6c, 0x61, 0x73, 0x74, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x12, 0x31, 0x0a, 0x0a,
	0x70, 0x75, 0x62, 0x6c, 0x69, 0x73, 0x68, 0x65, 0x72, 0x73, 0x18, 0x0a, 0x20, 0x03, 0x28, 0x0e,
	0x32, 0x11, 0x2e, 0x63, 0x6f, 0x6d, 0x69, 0x63, 0x73, 0x2e, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x73,
	0x68, 0x65, 0x72, 0x52, 0x0a, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x73, 0x68, 0x65, 0x72, 0x73, 0x12,
	0x25, 0x0a, 0x06, 0x67, 0x65, 0x6e, 0x72, 0x65, 0x73, 0x18, 0x0b, 0x20, 0x03, 0x28, 0x0e, 0x32,
	0x0d, 0x2e, 0x63, 0x6f, 0x6d, 0x69, 0x63, 0x73, 0x2e, 0x47, 0x65, 0x6e, 0x72, 0x65, 0x52, 0x06,
	0x67, 0x65, 0x6e, 0x72, 0x65, 0x73, 0x12, 0x26, 0x0a, 0x06, 0x72, 0x61, 0x74, 0x69, 0x6e, 0x67,
	0x18, 0x0c, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0e, 0x2e, 0x63, 0x6f, 0x6d, 0x69, 0x63, 0x73, 0x2e,
	0x52, 0x61, 0x74, 0x69, 0x6e, 0x67, 0x52, 0x06, 0x72, 0x61, 0x74, 0x69, 0x6e, 0x67, 0x12, 0x14,
	0x0a, 0x05, 0x74, 0x72, 0x61, 0x63, 0x6b, 0x18, 0x0d, 0x20, 0x01, 0x28, 0x08, 0x52, 0x05, 0x74,
	0x72, 0x61, 0x63, 0x6b, 0x12, 0x1f, 0x0a, 0x0b, 0x76, 0x69, 0x65, 0x77, 0x65, 0x64, 0x5f, 0x63,
	0x68, 0x61, 0x70, 0x18, 0x0e, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x0a, 0x76, 0x69, 0x65, 0x77, 0x65,
	0x64, 0x43, 0x68, 0x61, 0x70, 0x12, 0x18, 0x0a, 0x07, 0x64, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x64,
	0x18, 0x0f, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x64, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x64, 0x4a,
	0x04, 0x08, 0x10, 0x10, 0x15, 0x52, 0x05, 0x76, 0x69, 0x65, 0x77, 0x73, 0x2a, 0x5c, 0x0a, 0x09,
	0x43, 0x6f, 0x6d, 0x69, 0x63, 0x54, 0x79, 0x70, 0x65, 0x12, 0x10, 0x0a, 0x0c, 0x54, 0x59, 0x50,
	0x45, 0x5f, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12, 0x09, 0x0a, 0x05, 0x4d,
	0x41, 0x4e, 0x47, 0x41, 0x10, 0x01, 0x12, 0x0a, 0x0a, 0x06, 0x4d, 0x41, 0x4e, 0x48, 0x55, 0x41,
	0x10, 0x02, 0x12, 0x0a, 0x0a, 0x06, 0x4d, 0x41, 0x4e, 0x48, 0x57, 0x41, 0x10, 0x03, 0x12, 0x0b,
	0x0a, 0x07, 0x57, 0x45, 0x42, 0x54, 0x4f, 0x4f, 0x4e, 0x10, 0x03, 0x12, 0x09, 0x0a, 0x05, 0x4e,
	0x4f, 0x56, 0x45, 0x4c, 0x10, 0x04, 0x1a, 0x02, 0x10, 0x01, 0x2a, 0x4f, 0x0a, 0x06, 0x53, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x12, 0x12, 0x0a, 0x0e, 0x53, 0x54, 0x41, 0x54, 0x55, 0x53, 0x5f, 0x55,
	0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12, 0x0d, 0x0a, 0x09, 0x43, 0x4f, 0x4d, 0x50,
	0x4c, 0x45, 0x54, 0x45, 0x44, 0x10, 0x01, 0x12, 0x0a, 0x0a, 0x06, 0x4f, 0x4e, 0x5f, 0x41, 0x49,
	0x52, 0x10, 0x02, 0x12, 0x09, 0x0a, 0x05, 0x42, 0x52, 0x45, 0x41, 0x4b, 0x10, 0x03, 0x12, 0x0b,
	0x0a, 0x07, 0x44, 0x52, 0x4f, 0x50, 0x50, 0x45, 0x44, 0x10, 0x04, 0x2a, 0xf5, 0x01, 0x0a, 0x05,
	0x47, 0x65, 0x6e, 0x72, 0x65, 0x12, 0x11, 0x0a, 0x0d, 0x47, 0x45, 0x4e, 0x52, 0x45, 0x5f, 0x55,
	0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12, 0x0a, 0x0a, 0x06, 0x41, 0x43, 0x54, 0x49,
	0x4f, 0x4e, 0x10, 0x01, 0x12, 0x0d, 0x0a, 0x09, 0x41, 0x44, 0x56, 0x45, 0x4e, 0x54, 0x55, 0x52,
	0x45, 0x10, 0x02, 0x12, 0x0b, 0x0a, 0x07, 0x46, 0x41, 0x4e, 0x54, 0x41, 0x53, 0x59, 0x10, 0x03,
	0x12, 0x0f, 0x0a, 0x0b, 0x4f, 0x56, 0x45, 0x52, 0x50, 0x4f, 0x57, 0x45, 0x52, 0x45, 0x44, 0x10,
	0x04, 0x12, 0x0a, 0x0a, 0x06, 0x43, 0x4f, 0x4d, 0x45, 0x44, 0x59, 0x10, 0x05, 0x12, 0x09, 0x0a,
	0x05, 0x44, 0x52, 0x41, 0x4d, 0x41, 0x10, 0x06, 0x12, 0x0f, 0x0a, 0x0b, 0x53, 0x43, 0x48, 0x4f,
	0x4f, 0x4c, 0x5f, 0x4c, 0x49, 0x46, 0x45, 0x10, 0x07, 0x12, 0x0a, 0x0a, 0x06, 0x53, 0x59, 0x53,
	0x54, 0x45, 0x4d, 0x10, 0x08, 0x12, 0x10, 0x0a, 0x0c, 0x53, 0x55, 0x50, 0x45, 0x52, 0x4e, 0x41,
	0x54, 0x55, 0x52, 0x41, 0x4c, 0x10, 0x09, 0x12, 0x10, 0x0a, 0x0c, 0x4d, 0x41, 0x52, 0x54, 0x49,
	0x41, 0x4c, 0x5f, 0x41, 0x52, 0x54, 0x53, 0x10, 0x0a, 0x12, 0x0b, 0x0a, 0x07, 0x52, 0x4f, 0x4d,
	0x41, 0x4e, 0x43, 0x45, 0x10, 0x0b, 0x12, 0x0b, 0x0a, 0x07, 0x53, 0x48, 0x4f, 0x55, 0x4e, 0x45,
	0x4e, 0x10, 0x0c, 0x12, 0x11, 0x0a, 0x0d, 0x52, 0x45, 0x49, 0x4e, 0x43, 0x41, 0x52, 0x4e, 0x41,
	0x54, 0x49, 0x4f, 0x4e, 0x10, 0x0d, 0x12, 0x06, 0x0a, 0x02, 0x4f, 0x50, 0x10, 0x04, 0x12, 0x0f,
	0x0a, 0x0b, 0x43, 0x55, 0x4c, 0x54, 0x49, 0x56, 0x41, 0x54, 0x49, 0x4f, 0x4e, 0x10, 0x0a, 0x1a,
	0x02, 0x10, 0x01, 0x2a, 0xfd, 0x01, 0x0a, 0x09, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x73, 0x68, 0x65,
	0x72, 0x12, 0x15, 0x0a, 0x11, 0x50, 0x55, 0x42, 0x4c, 0x49, 0x53, 0x48, 0x45, 0x52, 0x5f, 0x55,
	0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12, 0x09, 0x0a, 0x05, 0x41, 0x53, 0x55, 0x52,
	0x41, 0x10, 0x01, 0x12, 0x10, 0x0a, 0x0c, 0x52, 0x45, 0x41, 0x50, 0x45, 0x52, 0x5f, 0x53, 0x43,
	0x41, 0x4e, 0x53, 0x10, 0x02, 0x12, 0x0f, 0x0a, 0x0b, 0x4d, 0x41, 0x4e, 0x48, 0x55, 0x41, 0x5f,
	0x50, 0x4c, 0x55, 0x53, 0x10, 0x03, 0x12, 0x0f, 0x0a, 0x0b, 0x46, 0x4c, 0x41, 0x4d, 0x45, 0x5f,
	0x53, 0x43, 0x41, 0x4e, 0x53, 0x10, 0x04, 0x12, 0x12, 0x0a, 0x0e, 0x4c, 0x55, 0x4d, 0x49, 0x4e,
	0x4f, 0x55, 0x53, 0x5f, 0x53, 0x43, 0x41, 0x4e, 0x53, 0x10, 0x05, 0x12, 0x0f, 0x0a, 0x0b, 0x52,
	0x45, 0x53, 0x45, 0x54, 0x5f, 0x53, 0x43, 0x41, 0x4e, 0x53, 0x10, 0x06, 0x12, 0x0f, 0x0a, 0x0b,
	0x49, 0x53, 0x45, 0x4b, 0x41, 0x49, 0x5f, 0x53, 0x43, 0x41, 0x4e, 0x10, 0x07, 0x12, 0x0f, 0x0a,
	0x0b, 0x52, 0x45, 0x41, 0x4c, 0x4d, 0x5f, 0x53, 0x43, 0x41, 0x4e, 0x53, 0x10, 0x08, 0x12, 0x12,
	0x0a, 0x0e, 0x4c, 0x45, 0x56, 0x49, 0x41, 0x54, 0x41, 0x4e, 0x5f, 0x53, 0x43, 0x41, 0x4e, 0x53,
	0x10, 0x09, 0x12, 0x0f, 0x0a, 0x0b, 0x4e, 0x49, 0x47, 0x48, 0x54, 0x5f, 0x53, 0x43, 0x41, 0x4e,
	0x53, 0x10, 0x0a, 0x12, 0x0e, 0x0a, 0x0a, 0x56, 0x4f, 0x49, 0x44, 0x5f, 0x53, 0x43, 0x41, 0x4e,
	0x53, 0x10, 0x0b, 0x12, 0x0f, 0x0a, 0x0b, 0x44, 0x52, 0x41, 0x4b, 0x45, 0x5f, 0x53, 0x43, 0x41,
	0x4e, 0x53, 0x10, 0x0c, 0x12, 0x0d, 0x0a, 0x09, 0x4e, 0x4f, 0x56, 0x45, 0x4c, 0x5f, 0x4d, 0x49,
	0x43, 0x10, 0x0d, 0x2a, 0xa8, 0x02, 0x0a, 0x06, 0x52, 0x61, 0x74, 0x69, 0x6e, 0x67, 0x12, 0x12,
	0x0a, 0x0e, 0x52, 0x41, 0x54, 0x49, 0x4e, 0x47, 0x5f, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e,
	0x10, 0x00, 0x12, 0x0b, 0x0a, 0x07, 0x46, 0x5f, 0x52, 0x41, 0x54, 0x45, 0x44, 0x10, 0x01, 0x12,
	0x0b, 0x0a, 0x07, 0x45, 0x5f, 0x52, 0x41, 0x54, 0x45, 0x44, 0x10, 0x02, 0x12, 0x0b, 0x0a, 0x07,
	0x44, 0x5f, 0x52, 0x41, 0x54, 0x45, 0x44, 0x10, 0x03, 0x12, 0x0b, 0x0a, 0x07, 0x43, 0x5f, 0x52,
	0x41, 0x54, 0x45, 0x44, 0x10, 0x04, 0x12, 0x0b, 0x0a, 0x07, 0x42, 0x5f, 0x52, 0x41, 0x54, 0x45,
	0x44, 0x10, 0x05, 0x12, 0x0b, 0x0a, 0x07, 0x41, 0x5f, 0x52, 0x41, 0x54, 0x45, 0x44, 0x10, 0x06,
	0x12, 0x0b, 0x0a, 0x07, 0x53, 0x5f, 0x52, 0x41, 0x54, 0x45, 0x44, 0x10, 0x07, 0x12, 0x0c, 0x0a,
	0x08, 0x53, 0x53, 0x5f, 0x52, 0x41, 0x54, 0x45, 0x44, 0x10, 0x08, 0x12, 0x0d, 0x0a, 0x09, 0x53,
	0x53, 0x53, 0x5f, 0x52, 0x41, 0x54, 0x45, 0x44, 0x10, 0x09, 0x12, 0x05, 0x0a, 0x01, 0x46, 0x10,
	0x01, 0x12, 0x0c, 0x0a, 0x08, 0x4f, 0x4e, 0x45, 0x5f, 0x53, 0x54, 0x41, 0x52, 0x10, 0x01, 0x12,
	0x05, 0x0a, 0x01, 0x45, 0x10, 0x02, 0x12, 0x05, 0x0a, 0x01, 0x44, 0x10, 0x03, 0x12, 0x0d, 0x0a,
	0x09, 0x54, 0x57, 0x4f, 0x5f, 0x53, 0x54, 0x41, 0x52, 0x53, 0x10, 0x03, 0x12, 0x05, 0x0a, 0x01,
	0x43, 0x10, 0x04, 0x12, 0x05, 0x0a, 0x01, 0x42, 0x10, 0x05, 0x12, 0x0f, 0x0a, 0x0b, 0x54, 0x48,
	0x52, 0x45, 0x45, 0x5f, 0x53, 0x54, 0x41, 0x52, 0x53, 0x10, 0x05, 0x12, 0x05, 0x0a, 0x01, 0x41,
	0x10, 0x06, 0x12, 0x05, 0x0a, 0x01, 0x53, 0x10, 0x07, 0x12, 0x0e, 0x0a, 0x0a, 0x46, 0x4f, 0x55,
	0x52, 0x5f, 0x53, 0x54, 0x41, 0x52, 0x53, 0x10, 0x07, 0x12, 0x06, 0x0a, 0x02, 0x53, 0x53, 0x10,
	0x08, 0x12, 0x07, 0x0a, 0x03, 0x53, 0x53, 0x53, 0x10, 0x09, 0x12, 0x0e, 0x0a, 0x0a, 0x46, 0x49,
	0x56, 0x45, 0x5f, 0x53, 0x54, 0x41, 0x52, 0x53, 0x10, 0x09, 0x1a, 0x02, 0x10, 0x01, 0x42, 0x08,
	0x5a, 0x03, 0x2f, 0x70, 0x62, 0x90, 0x01, 0x01, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_comics_proto_rawDescOnce sync.Once
	file_comics_proto_rawDescData = file_comics_proto_rawDesc
)

func file_comics_proto_rawDescGZIP() []byte {
	file_comics_proto_rawDescOnce.Do(func() {
		file_comics_proto_rawDescData = protoimpl.X.CompressGZIP(file_comics_proto_rawDescData)
	})
	return file_comics_proto_rawDescData
}

var file_comics_proto_enumTypes = make([]protoimpl.EnumInfo, 5)
var file_comics_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_comics_proto_goTypes = []any{
	(ComicType)(0),                // 0: comics.ComicType
	(Status)(0),                   // 1: comics.Status
	(Genre)(0),                    // 2: comics.Genre
	(Publisher)(0),                // 3: comics.Publisher
	(Rating)(0),                   // 4: comics.Rating
	(*Comics)(nil),                // 5: comics.Comics
	(*Comic)(nil),                 // 6: comics.Comic
	(*timestamppb.Timestamp)(nil), // 7: google.protobuf.Timestamp
}
var file_comics_proto_depIdxs = []int32{
	6, // 0: comics.Comics.comics:type_name -> comics.Comic
	0, // 1: comics.Comic.type:type_name -> comics.ComicType
	1, // 2: comics.Comic.status:type_name -> comics.Status
	7, // 3: comics.Comic.last_update:type_name -> google.protobuf.Timestamp
	3, // 4: comics.Comic.publishers:type_name -> comics.Publisher
	2, // 5: comics.Comic.genres:type_name -> comics.Genre
	4, // 6: comics.Comic.rating:type_name -> comics.Rating
	7, // [7:7] is the sub-list for method output_type
	7, // [7:7] is the sub-list for method input_type
	7, // [7:7] is the sub-list for extension type_name
	7, // [7:7] is the sub-list for extension extendee
	0, // [0:7] is the sub-list for field type_name
}

func init() { file_comics_proto_init() }
func file_comics_proto_init() {
	if File_comics_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_comics_proto_rawDesc,
			NumEnums:      5,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_comics_proto_goTypes,
		DependencyIndexes: file_comics_proto_depIdxs,
		EnumInfos:         file_comics_proto_enumTypes,
		MessageInfos:      file_comics_proto_msgTypes,
	}.Build()
	File_comics_proto = out.File
	file_comics_proto_rawDesc = nil
	file_comics_proto_goTypes = nil
	file_comics_proto_depIdxs = nil
}
