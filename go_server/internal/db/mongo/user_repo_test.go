package mongo

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"comics/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
)

const (
	failedToCreateUser = "Failed to create user"
	failedToUpdateUser = "Failed to update user"
	failedToDeleteUser = "Failed to delete user"
	dbName             = "test_comics_db"
	collName           = "test_users"
)

// TestUserRepository provides a test suite for the MongoDB user repository
type TestUserRepository struct {
	client     Client
	database   Database
	collection Collection
	userRepo   domain.UserStore
}

var (
	testClient Client
)

func TestMain(m *testing.M) {
	// Setup MongoDB container
	ctx := context.Background()
	mongoContainer, err := mongodb.Run(ctx, "mongo:latest")
	if err != nil {
		log.Fatalf("Failed to start MongoDB container: %v", err)
	}

	// Get connection string
	connectionString, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("Failed to get connection string: %v", err)
	}

	// Create custom MongoDB client
	testClient, err = NewMongoClient(ctx, nil, connectionString)
	if err != nil {
		log.Fatalf("Failed to create MongoDB client: %v", err)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	if err := testClient.Database(dbName).Collection(collName).Drop(ctx); err != nil {
		log.Fatalf("Failed to drop collection: %v", err)
	}
	if err := testClient.Disconnect(ctx); err != nil {
		log.Fatalf("Failed to disconnect from MongoDB: %v", err)
	}
	if err := mongoContainer.Terminate(ctx); err != nil {
		log.Fatalf("Failed to terminate MongoDB container: %v", err)
	}

	os.Exit(code)
}

func TestUserRepo(t *testing.T) {
	ctx := context.Background()
	// Connect to the client
	err := testClient.Connect(context.Background())
	require.NoError(t, err, "Failed to connect to MongoDB")

	// Create user repository
	userRepo := NewUserRepo(testClient, dbName, collName)

	// Prepare a test user
	testUser := &domain.User{
		ID:        uuid.New(),
		Username:  "testuser",
		Email:     "test@example.com",
		Password:  "hashedpassword",
		CreatedAt: time.Now(),
	}

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
	})

	t.Run("Get User By Username", func(t *testing.T) {
		retrievedUser, err := userRepo.GetByUsername(ctx, testUser.Username)
		assert.NoError(t, err)
		assert.NotNil(t, retrievedUser)
		if retrievedUser != nil {
			assert.Equal(t, testUser.Username, retrievedUser.Username)
		}
	})

	t.Run("Update User", func(t *testing.T) {
		testUser.Username = "updatedusername"
		err := userRepo.Update(ctx, testUser)
		assert.NoError(t, err)

		// Verify update
		retrievedUser, err := userRepo.GetByID(ctx, testUser.ID)
		assert.NoError(t, err)
		assert.Equal(t, "updatedusername", retrievedUser.Username)
	})

	t.Run("List Users", func(t *testing.T) {
		users, _, err := userRepo.List(ctx, 1, 10)
		assert.NoError(t, err)
		assert.NotEmpty(t, users)
	})

	t.Run("Delete User", func(t *testing.T) {
		err := userRepo.Delete(ctx, testUser.ID)
		assert.NoError(t, err)

		// Verify deletion
		_, err = userRepo.GetByID(ctx, testUser.ID)
		assert.Error(t, err)
	})
}

// setupTestEnvironment initializes the test database and repository
func setupTestEnvironment(_ *testing.T) domain.UserStore {
	return NewUserRepo(testClient, dbName, collName)
}

// TestCreateUser tests the user creation process
func TestCreateUser(t *testing.T) {
	uStore := setupTestEnvironment(t)

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
	err := uStore.Create(context.Background(), testUser)
	assert.NoError(t, err, failedToCreateUser)

	// Verify the user was created
	assert.NotEqual(t, uuid.Nil, testUser.ID, "User ID should be present")
	assert.True(t, testUser.CreatedAt.Before(time.Now()), "CreatedAt should be set")
	assert.True(t, testUser.Active, "User should be active by default")
}

// TestGetUserByID tests retrieving a user by ID
func TestGetUserByID(t *testing.T) {
	uStore := setupTestEnvironment(t)

	// Create a test user
	testUser := &domain.User{
		ID:        uuid.New(),
		Username:  "getbyiduser",
		Email:     "getbyid@example.com",
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	// Create the user
	err := uStore.Create(context.Background(), testUser)
	require.NoError(t, err, failedToCreateUser)

	// Retrieve the user by ID
	retrievedUser, err := uStore.GetByID(context.Background(), testUser.ID)
	assert.NoError(t, err, "Failed to get user by ID")
	assert.Equal(t, testUser.Username, retrievedUser.Username, "Retrieved user username should match")
	assert.Equal(t, testUser.Email, retrievedUser.Email, "Retrieved user email should match")
}

// TestGetUserByEmail tests retrieving a user by email
func TestGetUserByEmail(t *testing.T) {
	uStore := setupTestEnvironment(t)

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
	err := uStore.Create(context.Background(), testUser)
	require.NoError(t, err, failedToCreateUser)

	// Retrieve the user by email
	retrievedUser, err := uStore.GetByEmail(context.Background(), testUser.Email)
	assert.NoError(t, err, "Failed to get user by email")
	assert.Equal(t, testUser.Username, retrievedUser.Username, "Retrieved user username should match")
}

// TestUpdateUser tests updating a user
func TestUpdateUser(t *testing.T) {
	uStore := setupTestEnvironment(t)

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
	err := uStore.Create(context.Background(), testUser)
	require.NoError(t, err, failedToCreateUser)

	// Update the user
	testUser.Username = "updatedusername"
	testUser.Role = "user"
	err = uStore.Update(context.Background(), testUser)
	assert.NoError(t, err, failedToUpdateUser)

	// Retrieve and verify updates
	updatedUser, err := uStore.GetByID(context.Background(), testUser.ID)
	assert.NoError(t, err, "Failed to retrieve updated user")
	assert.Equal(t, "updatedusername", updatedUser.Username, "Username should be updated")
	assert.Equal(t, "user", updatedUser.Role, "Role should be updated")
}

// TestDeleteUser tests deleting a user
func TestDeleteUser(t *testing.T) {
	uStore := setupTestEnvironment(t)

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
	err := uStore.Create(context.Background(), testUser)
	require.NoError(t, err, failedToCreateUser)

	// Delete the user
	err = uStore.Delete(context.Background(), testUser.ID)
	assert.NoError(t, err, failedToDeleteUser)

	// Try to retrieve the deleted user
	_, err = uStore.GetByID(context.Background(), testUser.ID)
	assert.Error(t, err, "Should not be able to retrieve deleted user")
}

// TestListUsers tests listing users with pagination
func TestListUsers(t *testing.T) {
	uStore := setupTestEnvironment(t)

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
		err := uStore.Create(context.Background(), user)
		require.NoError(t, err, "Failed to create test user")
	}

	// List users
	listedUsers, total, err := uStore.List(context.Background(), 1, 10)
	assert.NoError(t, err, "Failed to list users")
	assert.True(t, total >= 2, "Should have at least 2 users")
	assert.GreaterOrEqual(t, len(listedUsers), 2, "Should return at least 2 users")
}

// TestFindActiveUsersByRole tests finding active users by role
func TestFindActiveUsersByRole(t *testing.T) {
	uStore := setupTestEnvironment(t)

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
		err := uStore.Create(context.Background(), user)
		require.NoError(t, err, "Failed to create test user")
	}

	// Find active admin users
	activeAdmins, err := uStore.FindActiveUsersByRole(context.Background(), "admin")
	assert.NoError(t, err, "Failed to find active users by role")

	// Verify results
	assert.Len(t, activeAdmins, 1, "Should find only 1 active admin")
	assert.Equal(t, "adminuser", activeAdmins[0].Username, "Should be the active admin user")
}
