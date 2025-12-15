package repository

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"control_page/internal/adaptor"
	"control_page/internal/model/enum"
)

const collectionUserRole = "user_role"

var _ adaptor.UserRoleRepository = (*UserRoleMongoRepository)(nil)

// UserRoleMongoDocument represents the MongoDB document structure for user-role relationships
type UserRoleMongoDocument struct {
	ID     primitive.ObjectID `bson:"_id,omitempty"`
	UserID primitive.ObjectID `bson:"user_id"`
	RoleID primitive.ObjectID `bson:"role_id"`
}

type UserRoleMongoRepository struct {
	userRoleCollection   *mongo.Collection
	permissionCollection *mongo.Collection
}

func NewUserRoleMongoRepository(db *mongo.Database) *UserRoleMongoRepository {
	return &UserRoleMongoRepository{
		userRoleCollection:   db.Collection(collectionUserRole),
		permissionCollection: db.Collection(collectionRolePermission),
	}
}

func (r *UserRoleMongoRepository) AssignRole(ctx context.Context, userID, roleID string) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	roleObjectID, err := primitive.ObjectIDFromHex(roleID)
	if err != nil {
		return errors.New("invalid role ID")
	}

	// Check if assignment already exists
	count, err := r.userRoleCollection.CountDocuments(ctx, bson.M{
		"user_id": userObjectID,
		"role_id": roleObjectID,
	})
	if err != nil {
		return err
	}
	if count > 0 {
		return nil // Assignment already exists
	}

	doc := UserRoleMongoDocument{
		UserID: userObjectID,
		RoleID: roleObjectID,
	}

	_, err = r.userRoleCollection.InsertOne(ctx, doc)
	return err
}

func (r *UserRoleMongoRepository) RemoveRole(ctx context.Context, userID, roleID string) error {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	roleObjectID, err := primitive.ObjectIDFromHex(roleID)
	if err != nil {
		return errors.New("invalid role ID")
	}

	_, err = r.userRoleCollection.DeleteOne(ctx, bson.M{
		"user_id": userObjectID,
		"role_id": roleObjectID,
	})
	return err
}

func (r *UserRoleMongoRepository) GetUserPermissions(ctx context.Context, userID string) ([]enum.Permission, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	// Use aggregation pipeline to get distinct permissions for user's roles
	pipeline := mongo.Pipeline{
		// Match user's roles
		{{Key: "$match", Value: bson.M{"user_id": userObjectID}}},
		// Lookup permissions for each role
		{{Key: "$lookup", Value: bson.M{
			"from":         collectionRolePermission,
			"localField":   "role_id",
			"foreignField": "role_id",
			"as":           "permissions",
		}}},
		// Unwind permissions array
		{{Key: "$unwind", Value: bson.M{"path": "$permissions"}}},
		// Group to get distinct permissions
		{{Key: "$group", Value: bson.M{
			"_id": "$permissions.permission",
		}}},
	}

	cursor, err := r.userRoleCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []struct {
		ID string `bson:"_id"`
	}
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	permissions := make([]enum.Permission, 0, len(results))
	for _, result := range results {
		permissions = append(permissions, enum.Permission(result.ID))
	}

	return permissions, nil
}
