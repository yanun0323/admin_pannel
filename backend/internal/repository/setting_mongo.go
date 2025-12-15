package repository

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"control_page/internal/adaptor"
	"control_page/internal/model"
)

const collectionSetting = "setting"

var _ adaptor.SettingRepository = (*SettingMongoRepository)(nil)

// SettingMongoDocument represents the MongoDB document structure for settings
type SettingMongoDocument struct {
	ID         primitive.ObjectID     `bson:"_id,omitempty"`
	Base       string                 `bson:"BASE"`
	Quote      string                 `bson:"QUOTE"`
	Strategy   string                 `bson:"STRATEGY"`
	Parameters bson.M                 `bson:"PARAMETERS"`
}

type SettingMongoRepository struct {
	collection *mongo.Collection
}

func NewSettingMongoRepository(db *mongo.Database) *SettingMongoRepository {
	return &SettingMongoRepository{
		collection: db.Collection(collectionSetting),
	}
}

func (r *SettingMongoRepository) GetAll(ctx context.Context) ([]model.Setting, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []SettingMongoDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	settings := make([]model.Setting, 0, len(docs))
	for _, doc := range docs {
		settings = append(settings, *documentToSetting(&doc))
	}

	return settings, nil
}

func (r *SettingMongoRepository) GetByID(ctx context.Context, id string) (*model.Setting, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, nil // Invalid ID format, return nil
	}

	var doc SettingMongoDocument
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return documentToSetting(&doc), nil
}

func (r *SettingMongoRepository) GetByBaseQuote(ctx context.Context, base, quote string) (*model.Setting, error) {
	var doc SettingMongoDocument
	err := r.collection.FindOne(ctx, bson.M{"BASE": base, "QUOTE": quote}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return documentToSetting(&doc), nil
}

func (r *SettingMongoRepository) Create(ctx context.Context, setting *model.Setting) error {
	doc := SettingMongoDocument{
		Base:       setting.Base,
		Quote:      setting.Quote,
		Strategy:   setting.Strategy,
		Parameters: convertParametersToBSON(setting.Parameters),
	}

	result, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		return err
	}

	objectID := result.InsertedID.(primitive.ObjectID)
	setting.MongoID = objectID.Hex()

	return nil
}

func (r *SettingMongoRepository) Update(ctx context.Context, setting *model.Setting) error {
	objectID, err := primitive.ObjectIDFromHex(setting.MongoID)
	if err != nil {
		return errors.New("invalid MongoDB ObjectID")
	}

	update := bson.M{
		"$set": bson.M{
			"BASE":       setting.Base,
			"QUOTE":      setting.Quote,
			"STRATEGY":   setting.Strategy,
			"PARAMETERS": convertParametersToBSON(setting.Parameters),
		},
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *SettingMongoRepository) UpdateParameters(ctx context.Context, id string, strategy string, parameters map[string]interface{}) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid MongoDB ObjectID")
	}

	// Update only the specific strategy parameters
	update := bson.M{
		"$set": bson.M{
			"PARAMETERS." + strategy: parameters,
		},
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *SettingMongoRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid MongoDB ObjectID")
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func documentToSetting(doc *SettingMongoDocument) *model.Setting {
	parameters := make(map[string]interface{})
	for key, value := range doc.Parameters {
		parameters[key] = value
	}

	return &model.Setting{
		MongoID:    doc.ID.Hex(),
		Base:       doc.Base,
		Quote:      doc.Quote,
		Strategy:   doc.Strategy,
		Parameters: parameters,
	}
}

func convertParametersToBSON(params map[string]interface{}) bson.M {
	result := bson.M{}
	for key, value := range params {
		result[key] = value
	}
	return result
}
