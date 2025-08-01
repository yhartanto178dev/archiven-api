package infrastructure

import (
	"context"
	"time"

	"github.com/yhartanto178dev/archiven-api/internal/auth/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserRepository struct {
	collection *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{
		collection: db.Collection("users"),
	}
}

func (r *UserRepository) FindByUsername(username string) (*domain.User, error) {
	var user domain.User
	err := r.collection.FindOne(context.Background(), bson.M{"username": username}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByEmail(email string) (*domain.User, error) {
	var user domain.User
	err := r.collection.FindOne(context.Background(), bson.M{"email": email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByID(id primitive.ObjectID) (*domain.User, error) {
	var user domain.User
	err := r.collection.FindOne(context.Background(), bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Create(user *domain.User) error {
	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.IsActive = true

	_, err := r.collection.InsertOne(context.Background(), user)
	return err
}

func (r *UserRepository) Update(user *domain.User) error {
	user.UpdatedAt = time.Now()
	_, err := r.collection.UpdateOne(
		context.Background(),
		bson.M{"_id": user.ID},
		bson.M{"$set": user},
	)
	return err
}

type RefreshTokenRepository struct {
	collection *mongo.Collection
}

func NewRefreshTokenRepository(db *mongo.Database) *RefreshTokenRepository {
	return &RefreshTokenRepository{
		collection: db.Collection("refresh_tokens"),
	}
}

func (r *RefreshTokenRepository) Store(token *domain.RefreshToken) error {
	_, err := r.collection.InsertOne(context.Background(), token)
	return err
}

func (r *RefreshTokenRepository) FindByToken(token string) (*domain.RefreshToken, error) {
	var refreshToken domain.RefreshToken
	err := r.collection.FindOne(context.Background(), bson.M{"token": token}).Decode(&refreshToken)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.ErrInvalidToken
		}
		return nil, err
	}
	return &refreshToken, nil
}

func (r *RefreshTokenRepository) RevokeToken(token string) error {
	_, err := r.collection.UpdateOne(
		context.Background(),
		bson.M{"token": token},
		bson.M{"$set": bson.M{"is_revoked": true}},
	)
	return err
}

func (r *RefreshTokenRepository) RevokeAllUserTokens(userID primitive.ObjectID) error {
	_, err := r.collection.UpdateMany(
		context.Background(),
		bson.M{"user_id": userID},
		bson.M{"$set": bson.M{"is_revoked": true}},
	)
	return err
}

func (r *RefreshTokenRepository) CleanupExpiredTokens() error {
	_, err := r.collection.DeleteMany(
		context.Background(),
		bson.M{"expires_at": bson.M{"$lt": time.Now()}},
	)
	return err
}
