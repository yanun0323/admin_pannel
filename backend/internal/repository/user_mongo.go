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

const collectionUser = "user"

var _ adaptor.UserRepository = (*UserMongoRepository)(nil)

// UserMongoDocument represents the MongoDB document structure for users
type UserMongoDocument struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	Username          string             `bson:"username"`
	Password          string             `bson:"password"`
	IsActive          bool               `bson:"is_active"`
	TOTPSecret        *string            `bson:"totp_secret,omitempty"`
	TOTPEnabled       bool               `bson:"totp_enabled"`
	PendingTOTPSecret *string            `bson:"pending_totp_secret,omitempty"`
	CreatedAt         time.Time          `bson:"created_at"`
	UpdatedAt         time.Time          `bson:"updated_at"`
}

type UserMongoRepository struct {
	collection *mongo.Collection
}

func NewUserMongoRepository(db *mongo.Database) *UserMongoRepository {
	return &UserMongoRepository{
		collection: db.Collection(collectionUser),
	}
}

func (r *UserMongoRepository) Create(ctx context.Context, user *model.User) error {
	now := time.Now()
	doc := UserMongoDocument{
		Username:          user.Username,
		Password:          user.Password,
		IsActive:          user.IsActive,
		TOTPSecret:        user.TOTPSecret,
		TOTPEnabled:       user.TOTPEnabled,
		PendingTOTPSecret: user.PendingTOTPSecret,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	result, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		return err
	}

	objectID := result.InsertedID.(primitive.ObjectID)
	user.ID = objectID.Hex()
	user.CreatedAt = now
	user.UpdatedAt = now

	return nil
}

func (r *UserMongoRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, nil // Invalid ID format
	}

	var doc UserMongoDocument
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return documentToUser(&doc), nil
}

func (r *UserMongoRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var doc UserMongoDocument
	err := r.collection.FindOne(ctx, bson.M{"username": username}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return documentToUser(&doc), nil
}

func (r *UserMongoRepository) Update(ctx context.Context, user *model.User) error {
	objectID, err := primitive.ObjectIDFromHex(user.ID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	user.UpdatedAt = time.Now()
	update := bson.M{
		"$set": bson.M{
			"username":   user.Username,
			"is_active":  user.IsActive,
			"updated_at": user.UpdatedAt,
		},
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *UserMongoRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid user ID")
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (r *UserMongoRepository) List(ctx context.Context) ([]model.User, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []UserMongoDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	users := make([]model.User, 0, len(docs))
	for _, doc := range docs {
		users = append(users, *documentToUser(&doc))
	}

	return users, nil
}

func (r *UserMongoRepository) UpdatePassword(ctx context.Context, id string, hashedPassword string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"$set": bson.M{
			"password":   hashedPassword,
			"updated_at": time.Now(),
		},
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *UserMongoRepository) UpdateUsername(ctx context.Context, id string, username string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"$set": bson.M{
			"username":   username,
			"updated_at": time.Now(),
		},
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *UserMongoRepository) UpdateRegistration(ctx context.Context, id string, hashedPassword, totpSecret string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"$set": bson.M{
			"password":    hashedPassword,
			"totp_secret": totpSecret,
			"updated_at":  time.Now(),
		},
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *UserMongoRepository) SetTOTPSecret(ctx context.Context, id string, secret string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"$set": bson.M{
			"totp_secret": secret,
			"updated_at":  time.Now(),
		},
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *UserMongoRepository) EnableTOTP(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"$set": bson.M{
			"totp_enabled": true,
			"updated_at":   time.Now(),
		},
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *UserMongoRepository) Activate(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"$set": bson.M{
			"is_active":  true,
			"updated_at": time.Now(),
		},
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *UserMongoRepository) SetPendingTOTPSecret(ctx context.Context, id string, secret string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"$set": bson.M{
			"pending_totp_secret": secret,
			"updated_at":          time.Now(),
		},
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *UserMongoRepository) ConfirmTOTPRebind(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid user ID")
	}

	// First, get the pending TOTP secret
	var doc UserMongoDocument
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&doc)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"totp_secret": doc.PendingTOTPSecret,
			"updated_at":  time.Now(),
		},
		"$unset": bson.M{
			"pending_totp_secret": "",
		},
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *UserMongoRepository) ClearPendingTOTPSecret(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid user ID")
	}

	update := bson.M{
		"$unset": bson.M{
			"pending_totp_secret": "",
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func documentToUser(doc *UserMongoDocument) *model.User {
	return &model.User{
		ID:                doc.ID.Hex(),
		Username:          doc.Username,
		Password:          doc.Password,
		IsActive:          doc.IsActive,
		TOTPSecret:        doc.TOTPSecret,
		TOTPEnabled:       doc.TOTPEnabled,
		PendingTOTPSecret: doc.PendingTOTPSecret,
		CreatedAt:         doc.CreatedAt,
		UpdatedAt:         doc.UpdatedAt,
	}
}
