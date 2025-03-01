package mongo

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"comics/domain"
	"comics/internal/repo"
	"comics/internal/tracer"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
)

var (
	userRepo  *UserRepo
	userDBcfg = DefaultConfig()
)

func TestMain(m *testing.M) {
	// Setup MongoDB container
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	mongoContainer, err := mongodb.Run(ctx, "mongo:latest",
		mongodb.WithReplicaSet("rs0"),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start MongoDB container")
	}

	// Get connection string
	testUri, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get connection string")
	}

	// Set DB config
	userDBcfg.Addr = testUri + "/?directConnection=true"
	userDBcfg.Name = "comics_db_test"
	userDBcfg.BackoffTimeout = 1 * time.Second

	// Create custom UserRepo
	userRepo, err = NewUserRepo(ctx, userDBcfg, &tracer.TracerConfig{
		ServiceName: "comics-service-test",
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create UserRepo")
	}

	// Run tests
	code := m.Run()

	// Cleanup
	if err := userRepo.Client().Database(userDBcfg.Name).
		Collection(userDBcfg.TableUsers).
		Drop(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to drop collection")
	}
	if err := userRepo.Close(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to disconnect from MongoDB")
	}
	if err := mongoContainer.Terminate(ctx); err != nil {
		log.Fatal().Err(err).Msg("Failed to terminate MongoDB container")
	}

	os.Exit(code)
}

func TestUserRepo(t *testing.T) {
	ctx := context.Background()
	startTime := time.Now()

	// Prepare valid test users
	users := []*domain.User{{
		ID:       uuid.New(),
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
		Role:     "user",
		Active:   false,
	}, {
		ID:       uuid.New(),
		Username: "adminuser",
		Email:    "admin@example.com",
		Password: "hashedpassword",
		Role:     "admin",
		Active:   true,
	}, {
		ID:       uuid.New(),
		Username: "inactiveadmin",
		Email:    "inactiveadmin@example.com",
		Role:     "admin",
		Active:   false,
	},
	}
	testUser := users[0]

	// Bad user
	badUser := &domain.User{ID: uuid.New()}

	t.Run("Create User", func(t *testing.T) {
		// Create users
		for _, user := range users {
			err := userRepo.Create(context.Background(), user)
			require.NoError(t, err, "Failed to create test user")

			// Verify the user was created
			assert.True(t, user.CreatedAt.After(startTime), "CreatedAt should be set")
			assert.True(t, user.UpdatedAt.After(startTime), "UpdatedAt should be set")
		}

		// Verify bad users
		err := userRepo.Create(ctx, nil)
		assert.Error(t, err, "Should not be able to create a user from nil")
		err = userRepo.Create(ctx, badUser)
		assert.Error(t, err, "Should not be able to create a user with no username or email")
	})

	t.Run("Get User By ID", func(t *testing.T) {
		retrievedUser, err := userRepo.GetByID(ctx, testUser.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedUser)
		if retrievedUser != nil {
			assert.Equal(t, testUser.Username, retrievedUser.Username)
		}
		// Verify bad queries
		nilUser, err := userRepo.GetByID(ctx, uuid.New())
		assert.Error(t, err)
		assert.Nil(t, nilUser, "Should be an inexistent user")
	})

	t.Run("Get User By Email", func(t *testing.T) {
		retrievedUser, err := userRepo.GetByEmail(ctx, testUser.Email)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedUser)
		if retrievedUser != nil {
			assert.Equal(t, testUser.Email, retrievedUser.Email)
		}
		// Verify bad queries
		nilUser, err := userRepo.GetByEmail(ctx, "nonexistent@example.com")
		assert.Error(t, err)
		assert.Nil(t, nilUser)

		badQueryUser, err := userRepo.GetByEmail(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, badQueryUser)
	})

	t.Run("Get User By Username", func(t *testing.T) {
		retrievedUser, err := userRepo.GetByUsername(ctx, testUser.Username)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedUser)
		if retrievedUser != nil {
			assert.Equal(t, testUser.Username, retrievedUser.Username)
		}
		// Verify bad queries
		nilUser, err := userRepo.GetByUsername(ctx, "nonexistentuser")
		assert.Error(t, err)
		assert.Nil(t, nilUser)

		badQueryUser, err := userRepo.GetByUsername(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, badQueryUser)
	})

	t.Run("Update User", func(t *testing.T) {
		testUser.Username = "updatedusername"
		err := userRepo.Update(ctx, testUser)
		assert.NoError(t, err)

		// Verify update
		retrievedUser, err := userRepo.GetByID(ctx, testUser.ID)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedUser)
		if retrievedUser != nil {
			assert.Equal(t, "updatedusername", retrievedUser.Username)
		}

		// Verify bad update
		err = userRepo.Update(ctx, badUser)
		assert.Error(t, err, "Should not be able to update an inexistent user")

		err = userRepo.Update(ctx, nil)
		assert.Error(t, err, "Should not be able to update a user from nil")
	})

	t.Run("List Users", func(t *testing.T) {
		users, total, err := userRepo.List(ctx, 1, 10)
		assert.NoError(t, err)
		assert.NotEmpty(t, users)
		assert.True(t, total >= 2)
		assert.GreaterOrEqual(t, len(users), 2)

		// Verify bad list
		_, _, err = userRepo.List(ctx, 0, 10)
		assert.Error(t, err)

		_, _, err = userRepo.List(ctx, 1, 0)
		assert.Error(t, err)
	})

	t.Run("Find Active Users By Role", func(t *testing.T) {
		activeAdmins, err := userRepo.FindActiveUsersByRole(ctx, "admin")
		assert.NoError(t, err, "Failed to find active users by role")

		// Verify results
		assert.Len(t, activeAdmins, 1, "Should find only 1 active admin")
		if len(activeAdmins) > 0 {
			assert.Equal(t, "admin", activeAdmins[0].Role, "Should find only admin users")
			assert.Equal(t, "adminuser", activeAdmins[0].Username, "Should be the active admin user")
		}
	})

	t.Run("Delete User", func(t *testing.T) {
		for _, user := range users {
			err := userRepo.Delete(ctx, user.ID)
			assert.NoError(t, err)
		}

		// Verify deletion
		_, err := userRepo.GetByID(ctx, testUser.ID)
		assert.Error(t, err)

		// Verify bad delete
		err = userRepo.Delete(ctx, badUser.ID)
		assert.Error(t, err)
	})

	t.Run("Transactions with Users", func(t *testing.T) {
		err := userRepo.Create(ctx, testUser)
		assert.NoError(t, err, "Shouldn't fail to create user")

		err = userRepo.Tx(ctx, func(ctx context.Context) error {
			testUser.Username = "newusername"
			err := userRepo.Update(ctx, testUser)
			assert.NoError(t, err, "Shouldn't fail to update user")
			testUser.Email = "newemail@example.com"
			err = userRepo.Update(ctx, testUser)
			assert.NoError(t, err, "Shouldn't fail to update user")
			return nil
		})
		assert.NoError(t, err, "Shouldn't fail to execute tx")

		updatedUser, err := userRepo.GetByID(ctx, testUser.ID)
		assert.NoError(t, err, "Shouldn't fail to get updated user")
		assert.Equal(t, "newusername", updatedUser.Username)
		assert.Equal(t, "newemail@example.com", updatedUser.Email)

		// Verify bad tx
		err = userRepo.Tx(ctx, func(ctx context.Context) error {
			err := userRepo.Delete(ctx, testUser.ID)
			assert.NoError(t, err, "Shouldn't fail to delete user")
			return errors.New("tx error")
		})
		assert.Error(t, err, "Should fail to execute tx")

		nonDeletedUser, err := userRepo.GetByID(ctx, testUser.ID)
		assert.NoError(t, err, "Shouldn't fail to get non deleted user")
		assert.NotNil(t, nonDeletedUser)
		assert.Equal(t, testUser.Username, nonDeletedUser.Username)
		assert.Equal(t, testUser.Email, nonDeletedUser.Email)
	})

	t.Run("Metrics", func(t *testing.T) {
		_ = userRepo.Client()
		log.Debug().Msgf("stats: %v", userRepo.GetStats())
		assert.Equal(t, uint64(1),
			userRepo.Metrics().GetSnapshot().TotalCreatedConnections)
	})
}

// TestClientConnection tests the client connection methods
func TestClientConnection(t *testing.T) {
	ctx := context.Background()

	_, err := NewUserRepo(ctx, nil, nil)
	assert.Error(t, err, "Creating UserRepo with nil config")

	_, err = NewUserRepo(ctx, &repo.DBConfig{}, nil)
	assert.Error(t, err, "Creating UserRepo with empty config")

	newUserRepo, err := NewUserRepo(ctx, userDBcfg, &tracer.TracerConfig{
		ServiceName: "comics-service_test",
	})
	assert.NoError(t, err, "Failed to create new UserRepo")

	err = newUserRepo.Client().WaitForConnection(1 * time.Second)
	assert.NoError(t, err, "Failed to wait for connection")

	err = newUserRepo.Client().WaitForConnection(1 * time.Nanosecond)
	assert.Error(t, err, "Should not be able to wait for connection")

	_, err = newUserRepo.Client().StartSession()
	assert.NoError(t, err, "Failed to start session")

	err = newUserRepo.Close(ctx)
	assert.NoError(t, err, "Failed to close connection")

	isOpen := newUserRepo.Client().IsConnected()
	assert.False(t, isOpen, "Connection should be closed")
}
