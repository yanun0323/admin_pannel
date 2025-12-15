package connection

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoClient wraps the MongoDB client
type MongoClient struct {
	Client   *mongo.Client
	Database *mongo.Database
}

// NewMongo creates a new MongoDB connection
func NewMongo(uri, database string) (*MongoClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("connect to mongodb: %w", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("ping mongodb: %w", err)
	}

	return &MongoClient{
		Client:   client,
		Database: client.Database(database),
	}, nil
}

// Close closes the MongoDB connection
func (m *MongoClient) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.Client.Disconnect(ctx)
}

// Collection returns a MongoDB collection
func (m *MongoClient) Collection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}
