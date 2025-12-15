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
	"control_page/internal/model/enum"
)

const collectionRole = "role"
const collectionRolePermission = "role_permission"

var _ adaptor.RoleRepository = (*RoleMongoRepository)(nil)

// RoleMongoDocument represents the MongoDB document structure for roles
type RoleMongoDocument struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Name        string             `bson:"name"`
	Description string             `bson:"description"`
	CreatedAt   time.Time          `bson:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"`
}

// RolePermissionMongoDocument represents the MongoDB document for role permissions
type RolePermissionMongoDocument struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	RoleID     primitive.ObjectID `bson:"role_id"`
	Permission string             `bson:"permission"`
}

type RoleMongoRepository struct {
	roleCollection       *mongo.Collection
	permissionCollection *mongo.Collection
}

func NewRoleMongoRepository(db *mongo.Database) *RoleMongoRepository {
	return &RoleMongoRepository{
		roleCollection:       db.Collection(collectionRole),
		permissionCollection: db.Collection(collectionRolePermission),
	}
}

func (r *RoleMongoRepository) Create(ctx context.Context, role *model.Role) error {
	now := time.Now()
	doc := RoleMongoDocument{
		Name:        role.Name,
		Description: role.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	result, err := r.roleCollection.InsertOne(ctx, doc)
	if err != nil {
		return err
	}

	objectID := result.InsertedID.(primitive.ObjectID)
	role.ID = objectID.Hex()
	role.CreatedAt = now
	role.UpdatedAt = now

	return nil
}

func (r *RoleMongoRepository) GetByID(ctx context.Context, id string) (*model.Role, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, nil // Invalid ID format
	}

	var doc RoleMongoDocument
	err = r.roleCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return documentToRole(&doc), nil
}

func (r *RoleMongoRepository) GetByName(ctx context.Context, name string) (*model.Role, error) {
	var doc RoleMongoDocument
	err := r.roleCollection.FindOne(ctx, bson.M{"name": name}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return documentToRole(&doc), nil
}

func (r *RoleMongoRepository) Update(ctx context.Context, role *model.Role) error {
	objectID, err := primitive.ObjectIDFromHex(role.ID)
	if err != nil {
		return errors.New("invalid role ID")
	}

	role.UpdatedAt = time.Now()
	update := bson.M{
		"$set": bson.M{
			"name":        role.Name,
			"description": role.Description,
			"updated_at":  role.UpdatedAt,
		},
	}

	_, err = r.roleCollection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	return err
}

func (r *RoleMongoRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid role ID")
	}

	// Delete role permissions first
	_, err = r.permissionCollection.DeleteMany(ctx, bson.M{"role_id": objectID})
	if err != nil {
		return err
	}

	// Delete the role
	_, err = r.roleCollection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

func (r *RoleMongoRepository) List(ctx context.Context) ([]model.Role, error) {
	cursor, err := r.roleCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []RoleMongoDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	roles := make([]model.Role, 0, len(docs))
	for _, doc := range docs {
		roles = append(roles, *documentToRole(&doc))
	}

	return roles, nil
}

func (r *RoleMongoRepository) GetRolesByUserID(ctx context.Context, userID string) ([]model.Role, error) {
	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	// First, get user's role IDs from user_role collection
	userRoleCollection := r.roleCollection.Database().Collection(collectionUserRole)

	cursor, err := userRoleCollection.Find(ctx, bson.M{"user_id": userObjectID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var userRoles []UserRoleMongoDocument
	if err := cursor.All(ctx, &userRoles); err != nil {
		return nil, err
	}

	if len(userRoles) == 0 {
		return []model.Role{}, nil
	}

	// Get role IDs
	roleIDs := make([]primitive.ObjectID, 0, len(userRoles))
	for _, ur := range userRoles {
		roleIDs = append(roleIDs, ur.RoleID)
	}

	// Get roles by IDs
	roleCursor, err := r.roleCollection.Find(ctx, bson.M{"_id": bson.M{"$in": roleIDs}})
	if err != nil {
		return nil, err
	}
	defer roleCursor.Close(ctx)

	var roleDocs []RoleMongoDocument
	if err := roleCursor.All(ctx, &roleDocs); err != nil {
		return nil, err
	}

	roles := make([]model.Role, 0, len(roleDocs))
	for _, doc := range roleDocs {
		roles = append(roles, *documentToRole(&doc))
	}

	return roles, nil
}

func (r *RoleMongoRepository) AddPermission(ctx context.Context, roleID string, permission enum.Permission) error {
	roleObjectID, err := primitive.ObjectIDFromHex(roleID)
	if err != nil {
		return errors.New("invalid role ID")
	}

	// Check if permission already exists
	count, err := r.permissionCollection.CountDocuments(ctx, bson.M{
		"role_id":    roleObjectID,
		"permission": permission.String(),
	})
	if err != nil {
		return err
	}
	if count > 0 {
		return nil // Permission already exists
	}

	doc := RolePermissionMongoDocument{
		RoleID:     roleObjectID,
		Permission: permission.String(),
	}

	_, err = r.permissionCollection.InsertOne(ctx, doc)
	return err
}

func (r *RoleMongoRepository) RemovePermission(ctx context.Context, roleID string, permission enum.Permission) error {
	roleObjectID, err := primitive.ObjectIDFromHex(roleID)
	if err != nil {
		return errors.New("invalid role ID")
	}

	_, err = r.permissionCollection.DeleteOne(ctx, bson.M{
		"role_id":    roleObjectID,
		"permission": permission.String(),
	})
	return err
}

func (r *RoleMongoRepository) GetPermissions(ctx context.Context, roleID string) ([]enum.Permission, error) {
	roleObjectID, err := primitive.ObjectIDFromHex(roleID)
	if err != nil {
		return nil, errors.New("invalid role ID")
	}

	cursor, err := r.permissionCollection.Find(ctx, bson.M{"role_id": roleObjectID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []RolePermissionMongoDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}

	permissions := make([]enum.Permission, 0, len(docs))
	for _, doc := range docs {
		permissions = append(permissions, enum.Permission(doc.Permission))
	}

	return permissions, nil
}

func (r *RoleMongoRepository) SetPermissions(ctx context.Context, roleID string, permissions []enum.Permission) error {
	roleObjectID, err := primitive.ObjectIDFromHex(roleID)
	if err != nil {
		return errors.New("invalid role ID")
	}

	// Delete existing permissions
	_, err = r.permissionCollection.DeleteMany(ctx, bson.M{"role_id": roleObjectID})
	if err != nil {
		return err
	}

	// Insert new permissions
	if len(permissions) == 0 {
		return nil
	}

	docs := make([]interface{}, 0, len(permissions))
	for _, p := range permissions {
		docs = append(docs, RolePermissionMongoDocument{
			RoleID:     roleObjectID,
			Permission: p.String(),
		})
	}

	_, err = r.permissionCollection.InsertMany(ctx, docs)
	return err
}

func documentToRole(doc *RoleMongoDocument) *model.Role {
	return &model.Role{
		ID:          doc.ID.Hex(),
		Name:        doc.Name,
		Description: doc.Description,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
	}
}
