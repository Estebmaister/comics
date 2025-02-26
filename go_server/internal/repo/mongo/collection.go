package mongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Database interface abstracts MongoDB database operations
type Database interface {
	Collection(name string, opts ...options.Lister[options.CollectionOptions]) Collection
	Client() Client
}

// Collection interface abstracts MongoDB collection operations
type Collection interface {
	InsertOne(ctx context.Context, document any, opts ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error)
	FindOne(ctx context.Context, filter any, opts ...options.Lister[options.FindOneOptions]) *mongo.SingleResult
	Find(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) (*mongo.Cursor, error)
	UpdateByID(ctx context.Context, id any, update any, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error)
	DeleteOne(ctx context.Context, filter any, opts ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error)
	CountDocuments(ctx context.Context, filter any, opts ...options.Lister[options.CountOptions]) (int64, error)
	Drop(ctx context.Context, opts ...options.Lister[options.DropCollectionOptions]) error
	Indexes() mongo.IndexView
}

type mongoDatabase struct {
	db *mongo.Database
}

type mongoCollection struct {
	coll *mongo.Collection
}

// Client interface abstracts MongoDB client operations
func (md *mongoDatabase) Client() Client {
	return &mongoClient{cl: md.db.Client()}
}

// Collection interface abstracts MongoDB collection operations
func (md *mongoDatabase) Collection(name string, opts ...options.Lister[options.CollectionOptions]) Collection {
	collection := md.db.Collection(name, opts...)
	return &mongoCollection{coll: collection}
}

// InsertOne interface abstracts MongoDB insert one operation
func (mc *mongoCollection) InsertOne(ctx context.Context, document any, opts ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error) {
	return mc.coll.InsertOne(ctx, document, opts[:]...)
}

// FindOne interface abstracts MongoDB find one operation
func (mc *mongoCollection) FindOne(ctx context.Context, filter any, opts ...options.Lister[options.FindOneOptions]) *mongo.SingleResult {
	return mc.coll.FindOne(ctx, filter, opts[:]...)
}

// Find interface abstracts MongoDB find operation
func (mc *mongoCollection) Find(ctx context.Context, filter any, opts ...options.Lister[options.FindOptions]) (*mongo.Cursor, error) {
	return mc.coll.Find(ctx, filter, opts[:]...)
}

// UpdateByID interface abstracts MongoDB update by ID operation
func (mc *mongoCollection) UpdateByID(ctx context.Context, id any, update any, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error) {
	return mc.coll.UpdateByID(ctx, id, update, opts[:]...)
}

// DeleteOne interface abstracts MongoDB delete one operation
func (mc *mongoCollection) DeleteOne(ctx context.Context, filter any, opts ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error) {
	return mc.coll.DeleteOne(ctx, filter, opts[:]...)
}

// CountDocuments interface abstracts MongoDB count documents operation
func (mc *mongoCollection) CountDocuments(ctx context.Context, filter any, opts ...options.Lister[options.CountOptions]) (int64, error) {
	return mc.coll.CountDocuments(ctx, filter, opts[:]...)
}

// Drop interface abstracts MongoDB drop operation
func (mc *mongoCollection) Drop(ctx context.Context, opts ...options.Lister[options.DropCollectionOptions]) error {
	return mc.coll.Drop(ctx, opts[:]...)
}

// Indexes interface abstracts MongoDB indexes operation
func (mc *mongoCollection) Indexes() mongo.IndexView {
	return mc.coll.Indexes()
}
