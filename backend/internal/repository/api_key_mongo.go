package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"control_page/internal/adaptor"
	"control_page/internal/model"
)

const collectionAPIToken = "api_token"

var _ adaptor.APIKeyRepository = (*APIKeyMongoRepository)(nil)

// APIKeyMongoDocument represents the MongoDB document structure
type APIKeyMongoDocument struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Name      string             `bson:"name"`
	Platform  string             `bson:"platform"`
	Enable    bool               `bson:"enable"`
	Testnet   bool               `bson:"testnet"`
	APIKey    string             `bson:"api_key"`
	APISecret string             `bson:"api_secret"`
}

type APIKeyMongoRepository struct {
	collection *mongo.Collection
}

func NewAPIKeyMongoRepository(db *mongo.Database) *APIKeyMongoRepository {
	return &APIKeyMongoRepository{
		collection: db.Collection(collectionAPIToken),
	}
}

func (r *APIKeyMongoRepository) Create(ctx context.Context, apiKey *model.APIKey) error {
	doc := APIKeyMongoDocument{
		Name:      apiKey.Name,
		Platform:  string(apiKey.Platform),
		Enable:    apiKey.IsActive,
		Testnet:   apiKey.IsTestnet,
		APIKey:    apiKey.APIKey,
		APISecret: apiKey.APISecret,
	}

	result, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		return err
	}

	// Set the generated ID back to the model
	objectID := result.InsertedID.(primitive.ObjectID)
	apiKey.ID = objectID.Hex()
	apiKey.CreatedAt = time.Now()
	apiKey.UpdatedAt = time.Now()

	return nil
}

func (r *APIKeyMongoRepository) GetByID(ctx context.Context, id string) (*model.APIKey, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, nil // Invalid ID format
	}

	var doc APIKeyMongoDocument
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return documentToAPIKey(&doc), nil
}

func (r *APIKeyMongoRepository) List(ctx context.Context) ([]model.APIKey, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []APIKeyMongoDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	apiKeys := make([]model.APIKey, 0, len(docs))
	for _, doc := range docs {
		apiKeys = append(apiKeys, *documentToAPIKey(&doc))
	}

	return apiKeys, nil
}

func (r *APIKeyMongoRepository) GetByPlatform(ctx context.Context, platform model.Platform) ([]model.APIKey, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"platform": string(platform)})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []APIKeyMongoDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	apiKeys := make([]model.APIKey, 0, len(docs))
	for _, doc := range docs {
		apiKeys = append(apiKeys, *documentToAPIKey(&doc))
	}

	return apiKeys, nil
}

func (r *APIKeyMongoRepository) Update(ctx context.Context, apiKey *model.APIKey) error {
	objectID, err := primitive.ObjectIDFromHex(apiKey.ID)
	if err != nil {
		return errors.New("invalid MongoDB ObjectID")
	}

	update := bson.M{
		"$set": bson.M{
			"name":       apiKey.Name,
			"api_key":    apiKey.APIKey,
			"api_secret": apiKey.APISecret,
			"testnet":    apiKey.IsTestnet,
			"enable":     apiKey.IsActive,
		},
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		return err
	}

	apiKey.UpdatedAt = time.Now()
	return nil
}

func (r *APIKeyMongoRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid MongoDB ObjectID")
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (r *APIKeyMongoRepository) GetActiveByPlatform(ctx context.Context, platform model.Platform, isTestnet bool) ([]model.APIKey, error) {
	cursor, err := r.collection.Find(ctx, bson.M{
		"platform": string(platform),
		"testnet":  isTestnet,
		"enable":   true,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []APIKeyMongoDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	apiKeys := make([]model.APIKey, 0, len(docs))
	for _, doc := range docs {
		apiKeys = append(apiKeys, *documentToAPIKey(&doc))
	}

	return apiKeys, nil
}

func documentToAPIKey(doc *APIKeyMongoDocument) *model.APIKey {
	return &model.APIKey{
		ID:        doc.ID.Hex(),
		Name:      doc.Name,
		Platform:  model.Platform(doc.Platform),
		APIKey:    doc.APIKey,
		APISecret: doc.APISecret,
		IsTestnet: doc.Testnet,
		IsActive:  doc.Enable,
		CreatedAt: doc.ID.Timestamp(),
		UpdatedAt: doc.ID.Timestamp(),
	}
}
