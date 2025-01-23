package mongo

import (
	"context"

	"comics/domain"
	"comics/mongo"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type userRepository struct {
	database   mongo.Database
	collection string
}

// NewUserRepo creates a new instance of a user repository
func NewUserRepo(db mongo.Database, collection string) domain.UserStore {
	return &userRepository{
		database:   db,
		collection: collection,
	}
}

// Fetch all users
func (ur *userRepository) Fetch(c context.Context) ([]domain.User, error) {
	collection := ur.database.Collection(ur.collection)
	opts := options.Find().SetProjection(bson.D{{Key: "password", Value: 0}})
	cursor, err := collection.Find(c, bson.D{}, opts)
	if err != nil {
		return nil, err
	}
	var users []domain.User
	err = cursor.All(c, &users)
	if users == nil {
		return users, err
	}
	return users, err
}

// GetByID retrieves a user by ID
func (ur *userRepository) GetByID(c context.Context, id string) (*domain.User, error) {
	collection := ur.database.Collection(ur.collection)
	var user *domain.User
	idHex, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return user, err
	}
	err = collection.FindOne(c, bson.M{"_id": idHex}).Decode(user)
	return user, err
}

// GetByEmail retrieves a user by email
func (ur *userRepository) GetByEmail(c context.Context, email string) (*domain.User, error) {
	collection := ur.database.Collection(ur.collection)
	var user *domain.User
	err := collection.FindOne(c, bson.M{"email": email}).Decode(user)
	return user, err
}

// GetByUsername retrieves a user by username
func (ur *userRepository) GetByUsername(c context.Context, username string) (*domain.User, error) {
	collection := ur.database.Collection(ur.collection)
	var user *domain.User
	err := collection.FindOne(c, bson.M{"username": username}).Decode(user)
	return user, err
}

// Create a new user
func (ur *userRepository) Create(c context.Context, user *domain.User) error {
	collection := ur.database.Collection(ur.collection)
	_, err := collection.InsertOne(c, user)
	return err
}

// Update a user by ID
func (ur *userRepository) Update(c context.Context, user *domain.User) error {
	collection := ur.database.Collection(ur.collection)
	_, err := collection.UpdateOne(c, bson.M{"_id": user.ID}, user)
	return err
}

// Delete a user by ID
func (ur *userRepository) Delete(c context.Context, id string) error {
	collection := ur.database.Collection(ur.collection)
	idHex, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = collection.DeleteOne(c, bson.M{"_id": idHex})
	return err
}
