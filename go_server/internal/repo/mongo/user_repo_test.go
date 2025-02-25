package mongo

import (
	"context"
	"os"
	"testing"
	"time"

	"comics/domain"
	"comics/internal/repo"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
)

const (
	failedToCreateUser = "Failed to create user"
	failedToUpdateUser = "Failed to update user"
	failedToDeleteUser = "Failed to delete user"
)

var (
	userRepo  *UserRepo
	userDBcfg = DefaultConfig()
)

func TestMain(m *testing.M) {
	// Setup MongoDB container
	ctx := context.Background()
	mongoContainer, err := mongodb.Run(ctx, "mongo:latest")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start MongoDB container")
	}

	// Get connection string
	testUri, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get connection string")
	}

	// Set DB config
	userDBcfg.Addr = testUri
	userDBcfg.Name = "comics_db_test"
	userDBcfg.BackoffTimeout = 1 * time.Second
	userDBcfg.TracerConfig.ServiceName += "_test"

	// Create custom UserRepo
	userRepo, err = NewUserRepo(ctx, userDBcfg)
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

	// Prepare a test user
	testUser := &domain.User{
		ID:       uuid.New(),
		Username: "testuser",
		Email:    "test@example.com",
		Password: "hashedpassword",
		Role:     "user",
		Active:   true,
	}

	// Bad user
	badUser := &domain.User{ID: uuid.New()}

	t.Run("Create User", func(t *testing.T) {
		err := userRepo.Create(ctx, testUser)
		assert.NoError(t, err)
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
		assert.Nil(t, nilUser)
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
		assert.Error(t, err)
	})

	t.Run("List Users", func(t *testing.T) {
		users, _, err := userRepo.List(ctx, 1, 10)
		assert.NoError(t, err)
		assert.NotEmpty(t, users)

		// Verify bad list
		_, _, err = userRepo.List(ctx, 0, 10)
		assert.Error(t, err)

		_, _, err = userRepo.List(ctx, 1, 0)
		assert.Error(t, err)
	})

	t.Run("Delete User", func(t *testing.T) {
		err := userRepo.Delete(ctx, testUser.ID)
		assert.NoError(t, err)

		// Verify deletion
		_, err = userRepo.GetByID(ctx, testUser.ID)
		assert.Error(t, err)

		// Verify bad delete
		err = userRepo.Delete(ctx, uuid.New())
		assert.Error(t, err)
	})

	t.Run("Metrics", func(t *testing.T) {
		_ = userRepo.Client()
		log.Debug().Msgf("stats: %v", userRepo.GetStats())
		assert.Equal(t, uint64(1),
			userRepo.Metrics().GetSnapshot().TotalCreatedConnections)
	})
}

// TestCreateUser tests the user creation process
func TestCreateUser(t *testing.T) {
	// Create a test user
	testUser := &domain.User{
		ID:        uuid.New(),
		Username:  "testuser",
		Email:     "test@example.com",
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	// Create the user
	err := userRepo.Create(context.Background(), testUser)
	assert.NoError(t, err, failedToCreateUser)

	// Verify the user was created
	assert.NotEqual(t, uuid.Nil, testUser.ID, "User ID should be present")
	assert.True(t, testUser.CreatedAt.Before(time.Now()), "CreatedAt should be set")
	assert.True(t, testUser.Active, "User should be active by default")
}

// TestGetUserByID tests retrieving a user by ID
func TestGetUserByID(t *testing.T) {

	// Create a test user
	testUser := &domain.User{
		ID:        uuid.New(),
		Username:  "ex_user",
		Email:     "getbyid@example.com",
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	// Create the user
	err := userRepo.Create(context.Background(), testUser)
	require.NoError(t, err, failedToCreateUser)

	// Retrieve the user by ID
	retrievedUser, err := userRepo.GetByID(context.Background(), testUser.ID)
	assert.NoError(t, err, "Failed to get user by ID")
	assert.Equal(t, testUser.Username, retrievedUser.Username, "Retrieved user username should match")
	assert.Equal(t, testUser.Email, retrievedUser.Email, "Retrieved user email should match")
}

// TestGetUserByEmail tests retrieving a user by email
func TestGetUserByEmail(t *testing.T) {

	// Create a test user
	testUser := &domain.User{
		ID:        uuid.New(),
		Username:  "getuserbyemail",
		Email:     "uniqueemail@example.com",
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	// Create the user
	err := userRepo.Create(context.Background(), testUser)
	require.NoError(t, err, failedToCreateUser)

	// Retrieve the user by email
	retrievedUser, err := userRepo.GetByEmail(context.Background(), testUser.Email)
	assert.NoError(t, err, "Failed to get user by email")
	assert.Equal(t, testUser.Username, retrievedUser.Username, "Retrieved user username should match")
}

// TestUpdateUser tests updating a user
func TestUpdateUser(t *testing.T) {

	// Create a test user
	testUser := &domain.User{
		ID:        uuid.New(),
		Username:  "updateuser",
		Email:     "update@example.com",
		Role:      "admin",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	// Create the user
	err := userRepo.Create(context.Background(), testUser)
	require.NoError(t, err, failedToCreateUser)

	// Update the user
	testUser.Username = "updatedusername"
	testUser.Role = "user"
	err = userRepo.Update(context.Background(), testUser)
	assert.NoError(t, err, failedToUpdateUser)

	// Retrieve and verify updates
	updatedUser, err := userRepo.GetByID(context.Background(), testUser.ID)
	assert.NoError(t, err, "Failed to retrieve updated user")
	assert.Equal(t, "updatedusername", updatedUser.Username, "Username should be updated")
	assert.Equal(t, "user", updatedUser.Role, "Role should be updated")
}

// TestDeleteUser tests deleting a user
func TestDeleteUser(t *testing.T) {

	// Create a test user
	testUser := &domain.User{
		ID:        uuid.New(),
		Username:  "deleteuser",
		Email:     "delete@example.com",
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	// Create the user
	err := userRepo.Create(context.Background(), testUser)
	require.NoError(t, err, failedToCreateUser)

	// Delete the user
	err = userRepo.Delete(context.Background(), testUser.ID)
	assert.NoError(t, err, failedToDeleteUser)

	// Try to retrieve the deleted user
	_, err = userRepo.GetByID(context.Background(), testUser.ID)
	assert.Error(t, err, "Should not be able to retrieve deleted user")
}

// TestListUsers tests listing users with pagination
func TestListUsers(t *testing.T) {

	// Create multiple test users
	users := []*domain.User{
		{
			ID:        uuid.New(),
			Username:  "user1",
			Email:     "user1@example.com",
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    true,
		},
		{
			ID:        uuid.New(),
			Username:  "user2",
			Email:     "user2@example.com",
			Role:      "user",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    true,
		},
	}

	// Create users
	for _, user := range users {
		err := userRepo.Create(context.Background(), user)
		require.NoError(t, err, "Failed to create test user")
	}

	// List users
	listedUsers, total, err := userRepo.List(context.Background(), 1, 10)
	assert.NoError(t, err, "Failed to list users")
	assert.True(t, total >= 2, "Should have at least 2 users")
	assert.GreaterOrEqual(t, len(listedUsers), 2, "Should return at least 2 users")
}

// TestFindActiveUsersByRole tests finding active users by role
func TestFindActiveUsersByRole(t *testing.T) {

	// Create test users with different roles
	users := []*domain.User{
		{
			ID:        uuid.New(),
			Username:  "adminuser",
			Email:     "admin@example.com",
			Role:      "admin",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    true,
		},
		{
			ID:        uuid.New(),
			Username:  "inactiveadmin",
			Email:     "inactiveadmin@example.com",
			Role:      "admin",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Active:    false,
		},
	}

	// Create users
	for _, user := range users {
		err := userRepo.Create(context.Background(), user)
		require.NoError(t, err, "Failed to create test user")
	}

	// Find active admin users
	activeAdmins, err := userRepo.FindActiveUsersByRole(context.Background(), "admin")
	assert.NoError(t, err, "Failed to find active users by role")

	// Verify results
	assert.Len(t, activeAdmins, 1, "Should find only 1 active admin")
	assert.Equal(t, "adminuser", activeAdmins[0].Username, "Should be the active admin user")
}

// TestClientConnection tests the client connection methods
func TestClientConnection(t *testing.T) {
	ctx := context.Background()

	_, err := NewUserRepo(ctx, nil)
	assert.Error(t, err, "Creating UserRepo with nil config")

	_, err = NewUserRepo(ctx, &repo.DBConfig{})
	assert.Error(t, err, "Creating UserRepo with empty config")

	userDBcfg.TracerConfig.ServiceName = "comics-service-test"
	newUserRepo, err := NewUserRepo(ctx, userDBcfg)
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
