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

const collectionSwitcher = "switcher"

var _ adaptor.SwitcherRepository = (*SwitcherMongoRepository)(nil)

type SwitcherMongoRepository struct {
	collection *mongo.Collection
}

func NewSwitcherMongoRepository(db *mongo.Database) *SwitcherMongoRepository {
	return &SwitcherMongoRepository{
		collection: db.Collection(collectionSwitcher),
	}
}

func (r *SwitcherMongoRepository) GetAll(ctx context.Context) ([]model.Switcher, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var switchers []model.Switcher
	for cursor.Next(ctx) {
		var raw bson.M
		if err := cursor.Decode(&raw); err != nil {
			return nil, err
		}

		switcher := documentToSwitcher(raw)
		switchers = append(switchers, *switcher)
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return switchers, nil
}

func (r *SwitcherMongoRepository) GetByID(ctx context.Context, id string) (*model.Switcher, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, nil // Invalid ID format, return nil
	}

	var raw bson.M
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&raw)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return documentToSwitcher(raw), nil
}

func (r *SwitcherMongoRepository) Create(ctx context.Context, switcher *model.Switcher) error {
	doc := bson.M{}
	for pair, config := range switcher.Pairs {
		doc[pair] = bson.M{"enable": config.Enable}
	}

	result, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		return err
	}

	objectID := result.InsertedID.(primitive.ObjectID)
	switcher.MongoID = objectID.Hex()

	return nil
}

func (r *SwitcherMongoRepository) Update(ctx context.Context, switcher *model.Switcher) error {
	objectID, err := primitive.ObjectIDFromHex(switcher.MongoID)
	if err != nil {
		return errors.New("invalid MongoDB ObjectID")
	}

	// Build update document
	update := bson.M{}
	for pair, config := range switcher.Pairs {
		update[pair] = bson.M{"enable": config.Enable}
	}

	_, err = r.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)
	return err
}

func (r *SwitcherMongoRepository) UpdatePair(ctx context.Context, id string, pair string, enable bool) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid MongoDB ObjectID")
	}

	_, err = r.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": bson.M{pair: bson.M{"enable": enable}}},
	)
	return err
}

func (r *SwitcherMongoRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid MongoDB ObjectID")
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func documentToSwitcher(raw bson.M) *model.Switcher {
	switcher := &model.Switcher{
		Pairs: make(map[string]model.SwitcherPair),
	}

	for key, value := range raw {
		if key == "_id" {
			if oid, ok := value.(primitive.ObjectID); ok {
				switcher.MongoID = oid.Hex()
			}
			continue
		}

		// Parse trading pair configuration
		if pairConfig, ok := value.(bson.M); ok {
			pair := model.SwitcherPair{}
			if enable, ok := pairConfig["enable"].(bool); ok {
				pair.Enable = enable
			}
			switcher.Pairs[key] = pair
		}
	}

	return switcher
}
