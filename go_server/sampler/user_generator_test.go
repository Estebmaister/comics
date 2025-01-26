package sampler

import (
	"reflect"
	"testing"
	"time"

	pb "comics/pkg/pb"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	newUserPlaceHolder = "newUser: %v"
)

func TestNewUser(t *testing.T) {
	t.Parallel()
	fakeUser := &pb.User{
		Id:          uuid.New().String(),
		Username:    "username",
		Email:       "email@domain.com",
		Password:    "BadPassword",
		Role:        pb.Role(pb.Role_value["USER"]),
		CreatedTime: timestamppb.New(time.Now()),
		UpdatedTime: timestamppb.New(time.Now()),
	}
	for i := 0; i < 1; i++ {
		newUser := NewUser()
		if reflect.DeepEqual(newUser, fakeUser) {
			t.Logf(newUserPlaceHolder, newUser)
			t.Error("NewUser() shouldn't be equal to fake")
		}
		if newUser.Username == "" {
			t.Logf(newUserPlaceHolder, newUser)
			t.Error("NewUser() should have a username")
		}
		if newUser.Password == "" {
			t.Logf(newUserPlaceHolder, newUser)
			t.Error("NewUser() should have a password")
		}
		if newUser.Email == "" {
			t.Logf(newUserPlaceHolder, newUser)
			t.Error("NewUser() should have an email")
		}

		if newUser.Role.String() != "USER" && newUser.Role.String() != "ADMIN" {
			t.Logf(newUserPlaceHolder, newUser)
			t.Error("NewUser() should have a role")
		}

		if newUser.UpdatedTime.Seconds < newUser.CreatedTime.Seconds {
			t.Logf(newUserPlaceHolder, newUser)
			t.Error("NewUser() should have a created time before updated time")
		}
	}
}
