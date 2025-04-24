package infrastructure

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yhartanto178dev/archiven-api/internal/archive/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

type ArchiveRepository struct {
	bucket *gridfs.Bucket
	client *mongo.Client
}

func NewArchiveRepository(client *mongo.Client, dbName string) (*ArchiveRepository, error) {
	bucketOpts := options.GridFSBucket().
		SetChunkSizeBytes(1024 * 1024).       // 1MB chunks
		SetWriteConcern(writeconcern.W1()).   // Faster writes with basic durability
		SetReadPreference(readpref.Primary()) // Read from primary for consistency

	bucket, err := gridfs.NewBucket(
		client.Database(dbName),
		bucketOpts,
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
	// Buat options untuk upload dengan metadata tambahan
	uploadOpts := options.GridFSUpload().
		SetMetadata(bson.M{
			"created_at": time.Now(),
			"is_temp":    false,
		})

	uploadStream, err := r.bucket.OpenUploadStream(file.Name, uploadOpts)
	if err != nil {
		return fmt.Errorf("failed to open upload stream: %v", err)
	}
	defer uploadStream.Close()

	size, err := uploadStream.Write(file.Content)
	if err != nil {
		return fmt.Errorf("failed to write file content: %v", err)
	}

	// Create archive object with metadata
	archive := &domain.Archive{
		ID:        uploadStream.FileID.(primitive.ObjectID),
		Name:      file.Name,
		Size:      int64(size),
		CreatedAt: time.Now(),
		IsTemp:    false,
	}
	archive.FormatSize()

	return nil
}

func (r *ArchiveRepository) FindByID(ctx context.Context, id string) (*domain.Archive, []byte, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid object ID: %v", err)
	}

	filter := bson.M{
		"_id": objID,
		"$or": []bson.M{
			{"deleted_at": nil},
			{"expires_at": bson.M{"$gt": time.Now()}},
		},
	}

	var results bson.M
	cursor, err := r.bucket.Find(filter)
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close(ctx)

	if !cursor.Next(ctx) {
		return nil, nil, domain.ErrArchiveNotFound
	}

	if err := cursor.Decode(&results); err != nil {
		return nil, nil, err
	}

	// Convert primitive.DateTime to time.Time
	uploadDate := time.Unix(int64(results["uploadDate"].(primitive.DateTime))/1000, 0)

	// Check if file is deleted
	var deletedAt *time.Time
	if val, ok := results["deleted_at"].(primitive.DateTime); ok {
		t := val.Time()
		deletedAt = &t
	}

	if deletedAt != nil {
		return nil, nil, domain.ErrArchiveNotFound
	}

	// Check if file has expired
	var expiresAt *time.Time
	if val, ok := results["expires_at"].(primitive.DateTime); ok {
		t := val.Time()
		expiresAt = &t
	}

	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return nil, nil, domain.ErrAlreadyExpire
	}

	// Handle potentially nil is_temp field
	isTemp := false // default value
	if tempVal, ok := results["is_temp"]; ok && tempVal != nil {
		if boolVal, ok := tempVal.(bool); ok {
			isTemp = boolVal
		}
	}

	archive := &domain.Archive{
		ID:        objID,
		Name:      results["filename"].(string),
		Size:      results["length"].(int64),
		CreatedAt: uploadDate,
		DeletedAt: deletedAt,
		ExpiresAt: expiresAt,
		IsTemp:    isTemp,
	}
	archive.FormatSize()

	// Download file content
	var buf bytes.Buffer
	_, err = r.bucket.DownloadToStream(objID, &buf)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download file: %v", err)
	}

	return archive, buf.Bytes(), nil
}

func (r *ArchiveRepository) FindAll(ctx context.Context, page, limit int) ([]domain.Archive, int64, error) {
	filter := bson.M{
		"deleted_at": nil,
		"$or": []bson.M{
			{"expires_at": nil},
			{"expires_at": bson.M{"$gt": time.Now()}},
		},
	}

	// Hitung total dokumen yang tidak terhapus
	total, err := r.bucket.GetFilesCollection().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	opts := options.Find().
		SetSkip(int64((page - 1) * limit)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cur, err := r.bucket.GetFilesCollection().Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}

	var files []bson.M
	if err = cur.All(ctx, &files); err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	var archives []domain.Archive
	for _, file := range files {
		archives = append(archives, mapToArchive(file))
	}

	return archives, total, nil
}

func (r *ArchiveRepository) DownloadFile(id primitive.ObjectID) (int64, []byte, error) {
	ctx := context.TODO()                        // Define ctx if not already defined
	archive, _, err := r.FindByID(ctx, id.Hex()) // Convert id to string using Hex()
	if err != nil {
		return 0, nil, err
	}

	// File sudah expired (temp delete)
	if archive.ExpiresAt != nil && archive.ExpiresAt.Before(time.Now()) {
		return 0, nil, domain.ErrAlreadyExpire
	}

	// File sudah di soft-delete
	if archive.DeletedAt != nil {
		return 0, nil, domain.ErrAlreadyDeleted
	}

	var buf bytes.Buffer
	_, err = r.bucket.DownloadToStream(archive.ID, &buf)
	if err != nil {
		return 0, nil, err
	}

	return 0, buf.Bytes(), nil
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

	cur, err := r.bucket.GetFilesCollection().Find(ctx, bson.M{
		"_id":        bson.M{"$in": objectIDs},
		"deleted_at": nil,
		"$or": []bson.M{
			{"expires_at": nil},
			{"expires_at": bson.M{"$gt": time.Now()}},
		},
	})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var files []bson.M
	if err = cur.All(ctx, &files); err != nil {
		return nil, err
	}

	var archives []domain.Archive
	for _, file := range files {
		archives = append(archives, mapToArchive(file))
	}

	return archives, nil
}

func (r *ArchiveRepository) Delete(ctx context.Context, id string, deleteType domain.DeleteType) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	switch deleteType {
	case domain.SoftDelete:
		return r.softDelete(ctx, objID)
	case domain.HardDelete:
		return r.hardDelete(ctx, objID)
	case domain.TempDelete:
		return r.tempDelete(ctx, objID)
	default:
		return domain.ErrDeleteNotAllowed
	}
}

func (r *ArchiveRepository) softDelete(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now()
	_, err := r.bucket.GetFilesCollection().UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"deleted_at": now}},
	)
	return err
}

func (r *ArchiveRepository) hardDelete(_ context.Context, id primitive.ObjectID) error {
	return r.bucket.Delete(id)
}

func (r *ArchiveRepository) tempDelete(ctx context.Context, id primitive.ObjectID) error {
	expiresAt := time.Now().Add(24 * time.Hour)
	_, err := r.bucket.GetFilesCollection().UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{
			"is_temp":    true,
			"expires_at": expiresAt,
		}},
	)
	return err
}

func (r *ArchiveRepository) Restore(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	result := r.bucket.GetFilesCollection().FindOneAndUpdate(
		ctx,
		bson.M{"_id": objID},
		bson.M{"$unset": bson.M{
			"deleted_at": "",
			"expires_at": "",
			"is_temp":    "",
		}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return domain.ErrArchiveNotFound
		}
		return result.Err()
	}
	return nil
}

func (r *ArchiveRepository) Exists(ctx context.Context, id string) (bool, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return false, err
	}

	count, err := r.bucket.GetFilesCollection().CountDocuments(
		ctx,
		bson.M{"_id": objID},
	)
	return count > 0, err
}

func (r *ArchiveRepository) DeleteExpiredTempFiles(ctx context.Context) error {
	_, err := r.bucket.GetFilesCollection().DeleteMany(
		ctx,
		bson.M{
			"is_temp":    true,
			"expires_at": bson.M{"$lt": time.Now()},
		},
	)
	return err
}

func mapToArchive(file bson.M) domain.Archive {
	var expiresAt *time.Time
	if val, ok := file["expires_at"].(primitive.DateTime); ok {
		t := val.Time()
		expiresAt = &t
	}

	var deletedAt *time.Time
	if val, ok := file["deleted_at"].(primitive.DateTime); ok {
		t := val.Time()
		deletedAt = &t
	}

	// Handle potentially nil is_temp field
	isTemp := false
	if val, ok := file["is_temp"]; ok && val != nil {
		if boolVal, ok := val.(bool); ok {
			isTemp = boolVal
		}
	}

	return domain.Archive{
		ID:        file["_id"].(primitive.ObjectID),
		Name:      file["filename"].(string),
		Size:      file["length"].(int64),
		CreatedAt: file["uploadDate"].(primitive.DateTime).Time(),
		DeletedAt: deletedAt,
		ExpiresAt: expiresAt,
		IsTemp:    isTemp,
	}
}
