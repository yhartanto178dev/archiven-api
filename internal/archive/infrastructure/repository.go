package infrastructure

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/yhartanto178dev/archiven-api/internal/archive/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ArchiveRepository struct {
	bucket *gridfs.Bucket
	client *mongo.Client
}

func NewArchiveRepository(client *mongo.Client, dbName string) (*ArchiveRepository, error) {
	bucket, err := gridfs.NewBucket(
		client.Database(dbName),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gridfs bucket: %v", err)
	}

	return &ArchiveRepository{
		bucket: bucket,
		client: client,
	}, nil
}

func (r *ArchiveRepository) Save(ctx context.Context, file domain.FileContent) error {
	uploadStream, err := r.bucket.OpenUploadStream(file.Name)
	if err != nil {
		return err
	}
	defer uploadStream.Close()

	// size, err := uploadStream.Write(file.Content)
	// if err != nil {
	// 	return  err
	// }

	// archive := &domain.Archive{
	// 	ID:        uploadStream.FileID.(primitive.ObjectID).Hex(),
	// 	Name:      file.Name,
	// 	Size:      int64(size),
	// 	CreatedAt: time.Now(),
	// }

	return nil
}

func (r *ArchiveRepository) FindByID(ctx context.Context, id string) (*domain.Archive, []byte, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid object ID: %v", err)
	}

	var results bson.M
	cursor, err := r.bucket.Find(bson.M{"_id": objID})
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close(ctx)

	if !cursor.Next(ctx) {
		return nil, nil, mongo.ErrNoDocuments
	}

	if err := cursor.Decode(&results); err != nil {
		return nil, nil, err
	}

	archive := &domain.Archive{
		ID:        id,
		Name:      results["filename"].(string),
		Size:      results["length"].(int64),
		CreatedAt: results["uploadDate"].(time.Time),
	}

	var buf []byte
	_, buf, err = r.downloadFile(objID)
	if err != nil {
		return nil, nil, err
	}

	return archive, buf, nil
}

func (r *ArchiveRepository) FindAll(ctx context.Context, page, limit int) ([]domain.Archive, int64, error) {
	skip := (page - 1) * limit
	skipValue := int64(skip)
	limitValue := int64(limit)
	opts := options.GridFSFind().SetSkip(int32(skipValue)).SetLimit(int32(limitValue))

	cursor, err := r.bucket.Find(bson.M{}, opts)
	if err != nil {
		return nil, 0, err
	}

	var files []bson.M
	if err = cursor.All(ctx, &files); err != nil {
		return nil, 0, err
	}

	total, err := r.bucket.GetFilesCollection().CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	var archives []domain.Archive
	for _, file := range files {
		uploadDate := time.Unix(int64(file["uploadDate"].(primitive.DateTime))/1000, 0)

		archives = append(archives, domain.Archive{
			ID:        file["_id"].(primitive.ObjectID).Hex(),
			Name:      file["filename"].(string),
			Size:      file["length"].(int64),
			CreatedAt: uploadDate,
		})
	}

	return archives, total, nil
}

func (r *ArchiveRepository) downloadFile(id primitive.ObjectID) (int64, []byte, error) {
	var buf bytes.Buffer
	size, err := r.bucket.DownloadToStream(id, &buf)
	if err != nil {
		return 0, nil, err
	}
	return size, buf.Bytes(), nil
}

// Tambahkan implementasi repository
func (r *ArchiveRepository) FindByIDs(ctx context.Context, ids []string) ([]domain.Archive, error) {
	var objectIDs []primitive.ObjectID
	for _, id := range ids {
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return nil, err
		}
		objectIDs = append(objectIDs, objID)
	}

	cursor, err := r.bucket.Find(bson.M{"_id": bson.M{"$in": objectIDs}})
	if err != nil {
		return nil, err
	}

	var files []bson.M
	if err = cursor.All(ctx, &files); err != nil {
		return nil, err
	}

	var archives []domain.Archive
	for _, file := range files {
		// Convert primitive.DateTime to time.Time
		uploadDate := time.Unix(int64(file["uploadDate"].(primitive.DateTime))/1000, 0)

		archives = append(archives, domain.Archive{
			ID:        file["_id"].(primitive.ObjectID).Hex(),
			Name:      file["filename"].(string),
			Size:      file["length"].(int64),
			CreatedAt: uploadDate,
		})
	}

	return archives, nil
}
