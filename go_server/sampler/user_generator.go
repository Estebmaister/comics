package sampler

import (
	pb "comics/pkg/pb"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	passLength = 20
)

// NewUser generates a new user with random values
func NewUser() *pb.User {
	createdTime := randomTimestamp()
	return &pb.User{
		Id:          NewUserID(),
		Email:       NewEmail(),
		Password:    NewPassword(),
		Username:    NewUsername(),
		Role:        NewRole(),
		CreatedTime: timestamppb.New(createdTime),
		UpdatedTime: timestamppb.New(randomTimestampSince(createdTime)),
	}
}

// NewUserID generates a new user ID
func NewUserID() string {
	return uuid.New().String()
}

// NewEmail generates a new email
func NewEmail() string {
	return randomString() + "@" +
		randomStringFromSet("email", "gmail", "yahoo", "hotmail", "outlook") +
		randomStringFromSet(".com", ".org", ".net", ".edu", ".info", ".biz")
}

// NewPassword generates a new password
func NewPassword() string {
	return randomStringOfLength(passLength)
}

// NewUsername generates a new username
func NewUsername() string {
	return randomString()
}

// NewRole generates a new role
func NewRole() pb.Role {
	return pb.Role(randomUInt(0, len(pb.Role_name))) // #nosec G115
}
