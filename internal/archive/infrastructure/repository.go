package infrastructure

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
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
			return nil, fmt.Errorf("invalid object ID %s: %v", id, err)
		}
		objectIDs = append(objectIDs, objID)
	}

	// Build filter for files collection
	filter := bson.M{
		"_id": bson.M{"$in": objectIDs},
		"$or": []bson.M{
			{"metadata.deleted_at": nil},
			{"metadata.deleted_at": bson.M{"$exists": false}},
		},
	}

	// Find files with projection to include necessary fields
	opts := options.Find().
		SetSort(bson.D{{Key: "uploadDate", Value: -1}}).
		SetProjection(bson.D{
			{Key: "_id", Value: 1},
			{Key: "filename", Value: 1},
			{Key: "length", Value: 1},
			{Key: "uploadDate", Value: 1},
			{Key: "metadata", Value: 1},
		})

	cur, err := r.bucket.GetFilesCollection().Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find documents: %v", err)
	}
	defer cur.Close(ctx)

	var files []bson.M
	if err = cur.All(ctx, &files); err != nil {
		return nil, fmt.Errorf("failed to decode documents: %v", err)
	}

	if len(files) == 0 {
		return nil, domain.ErrArchiveNotFound
	}

	// Map results and maintain order
	idMap := make(map[string]domain.Archive)
	for _, file := range files {
		archive := mapToArchive(file)
		idMap[archive.ID.Hex()] = archive
	}

	// Preserve requested order
	var archives []domain.Archive
	for _, id := range ids {
		if archive, ok := idMap[id]; ok {
			archives = append(archives, archive)
		}
	}

	if len(archives) == 0 {
		return nil, domain.ErrArchiveNotFound
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

func (r *ArchiveRepository) RestoreArchive(ctx context.Context, id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid object ID: %v", err)
	}

	// Find the file first with proper filter
	filter := bson.M{
		"_id":        objID,
		"deleted_at": bson.M{"$exists": true, "$ne": nil},
	}

	var result bson.M
	err = r.bucket.GetFilesCollection().FindOne(ctx, filter).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.ErrNotDeleted
		}
		return fmt.Errorf("failed to find document: %v", err)
	}

	now := time.Now()
	changeLog := domain.ChangeLog{
		Timestamp: now,
		Action:    "restore",
		UserID:    result["metadata"].(bson.M)["owner_id"].(string),
		Changes: []domain.Change{
			{
				Field:    "status",
				OldValue: "deleted",
				NewValue: "active",
			},
		},
	}

	// Get existing change logs and append new one
	var changeLogs []domain.ChangeLog
	if logs, ok := result["metadata"].(bson.M)["change_logs"].(primitive.A); ok {
		for _, l := range logs {
			if log, ok := l.(bson.M); ok {
				changeLogs = append(changeLogs, domain.ChangeLog{
					Timestamp: log["timestamp"].(primitive.DateTime).Time(),
					Action:    log["action"].(string),
					UserID:    log["user_id"].(string),
				})
			}
		}
	}
	changeLogs = append(changeLogs, changeLog)

	// Update document - remove deleted_at field
	update := bson.M{
		"$set": bson.M{
			"metadata.change_logs": changeLogs,
			"metadata.updated_at":  now,
		},
		"$unset": bson.M{
			"deleted_at": "",
		},
	}

	_, err = r.bucket.GetFilesCollection().UpdateOne(
		ctx,
		bson.M{"_id": objID},
		update,
	)
	if err != nil {
		return fmt.Errorf("failed to update document: %v", err)
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
	metadata, ok := file["metadata"].(bson.M)
	if !ok {
		// Return empty archive with basic fields if no metadata
		return domain.Archive{
			ID:        file["_id"].(primitive.ObjectID),
			Name:      file["filename"].(string),
			Size:      file["length"].(int64),
			CreatedAt: file["uploadDate"].(primitive.DateTime).Time(),
		}
	}

	// Extract metadata fields
	archive := domain.Archive{
		ID:          file["_id"].(primitive.ObjectID),
		Name:        file["filename"].(string),
		Size:        file["length"].(int64),
		Category:    metadata["category"].(string),
		Type:        metadata["type"].(string),
		Description: metadata["description"].(string),
		OwnerID:     metadata["owner_id"].(string),
		Version:     int(metadata["version"].(int32)),
		CreatedAt:   metadata["created_at"].(primitive.DateTime).Time(),
		UpdatedAt:   metadata["updated_at"].(primitive.DateTime).Time(),
		IsTemp:      metadata["is_temp"].(bool),
	}

	// Handle optional arrays
	if tags, ok := metadata["tags"].(primitive.A); ok {
		archive.Tags = make([]string, len(tags))
		for i, tag := range tags {
			archive.Tags[i] = tag.(string)
		}
	}

	// Handle optional ChangeLogs
	if changeLogs, ok := metadata["change_logs"].(primitive.A); ok {
		archive.ChangeLogs = make([]domain.ChangeLog, len(changeLogs))
		for i, cl := range changeLogs {
			if logMap, ok := cl.(bson.M); ok {
				archive.ChangeLogs[i] = domain.ChangeLog{
					Timestamp: logMap["timestamp"].(primitive.DateTime).Time(),
					Action:    logMap["action"].(string),
					UserID:    logMap["user_id"].(string),
				}
			}
		}
	}

	archive.FormatSize()
	return archive
}

// /versioning kategori
func (r *ArchiveRepository) FindExistingArchive(ctx context.Context, archive domain.Archive) (*domain.Archive, error) {
	// Find by filename only first
	filter := bson.M{
		"filename": archive.Name,
		"$or": []bson.M{
			{"metadata.deleted_at": nil},
			{"metadata.deleted_at": bson.M{"$exists": false}},
		},
	}

	cursor, err := r.bucket.Find(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find existing archive: %v", err)
	}
	defer cursor.Close(ctx)

	var latest *domain.Archive
	// Compare file content hash if needed
	for cursor.Next(ctx) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode result: %v", err)
		}

		current := mapToArchive(result)
		if latest == nil || current.Version > latest.Version {
			latest = &current
		}
	}

	return latest, nil
}

func (r *ArchiveRepository) SaveWithVersioning(ctx context.Context, archive domain.Archive, content []byte) (*domain.Archive, error) {
	existing, err := r.FindExistingArchive(ctx, archive)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	archive.UpdatedAt = now

	if existing != nil {
		// Always increment version for new uploads
		archive.ID = existing.ID
		archive.Version = existing.Version + 1
		archive.CreatedAt = existing.CreatedAt

		// Track changes
		changeLog := domain.ChangeLog{
			Timestamp: now,
			Action:    "update",
			UserID:    archive.OwnerID,
			Changes: []domain.Change{
				{
					Field:    "version",
					OldValue: existing.Version,
					NewValue: archive.Version,
				},
			},
		}

		// Add metadata changes to changelog
		if existing.Category != archive.Category {
			changeLog.Changes = append(changeLog.Changes, domain.Change{
				Field:    "category",
				OldValue: existing.Category,
				NewValue: archive.Category,
			})
		}
		if existing.Description != archive.Description {
			changeLog.Changes = append(changeLog.Changes, domain.Change{
				Field:    "description",
				OldValue: existing.Description,
				NewValue: archive.Description,
			})
		}
		if !reflect.DeepEqual(existing.Tags, archive.Tags) {
			changeLog.Changes = append(changeLog.Changes, domain.Change{
				Field:    "tags",
				OldValue: fmt.Sprintf("%v", existing.Tags),
				NewValue: fmt.Sprintf("%v", archive.Tags),
			})
		}

		archive.ChangeLogs = append(existing.ChangeLogs, changeLog)

		// Delete old version
		if err := r.bucket.Delete(existing.ID); err != nil {
			return nil, fmt.Errorf("failed to delete old version: %v", err)
		}
	} else {
		// Create new file
		archive.ID = primitive.NewObjectID()
		archive.Version = 1
		archive.CreatedAt = now
		archive.ChangeLogs = []domain.ChangeLog{
			{
				Timestamp: now,
				Action:    "upload",
				UserID:    archive.OwnerID,
				Changes:   []domain.Change{},
			},
		}
	}

	// Upload file with all metadata
	metadata := bson.D{
		{Key: "filename", Value: archive.Name},
		{Key: "category", Value: archive.Category},
		{Key: "type", Value: archive.Type},
		{Key: "tags", Value: archive.Tags},
		{Key: "description", Value: archive.Description},
		{Key: "owner_id", Value: archive.OwnerID},
		{Key: "version", Value: archive.Version},
		{Key: "created_at", Value: archive.CreatedAt},
		{Key: "updated_at", Value: now},
		{Key: "is_temp", Value: archive.IsTemp},
		{Key: "change_logs", Value: archive.ChangeLogs},
	}

	uploadOpts := options.GridFSUpload().SetMetadata(metadata)
	uploadStream, err := r.bucket.OpenUploadStream(
		archive.Name,
		uploadOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open upload stream: %v", err)
	}
	defer uploadStream.Close()

	size, err := uploadStream.Write(content)
	if err != nil {
		return nil, fmt.Errorf("failed to write file content: %v", err)
	}

	archive.Size = int64(size)
	archive.FormatSize()

	return &archive, nil
}

func (r *ArchiveRepository) GetHistory(ctx context.Context, id string) (*domain.History, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid object ID: %v", err)
	}

	var result bson.M
	err = r.bucket.GetFilesCollection().FindOne(ctx, bson.M{"_id": objID}).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrArchiveNotFound
		}
		return nil, fmt.Errorf("failed to find document: %v", err)
	}

	// Extract metadata
	metadata, ok := result["metadata"].(bson.M)
	if !ok {
		return nil, fmt.Errorf("invalid metadata format")
	}

	history := domain.History{
		ID:       id,
		FileName: result["filename"].(string),
		Logs:     make([]domain.HistoryEntry, 0),
	}

	// Extract change logs from metadata
	if changeLogs, exists := metadata["change_logs"]; exists && changeLogs != nil {
		if logs, ok := changeLogs.(primitive.A); ok {
			for _, logInterface := range logs {
				if log, ok := logInterface.(bson.M); ok {
					entry := domain.HistoryEntry{
						Timestamp: log["timestamp"].(primitive.DateTime).Time(),
						Action:    log["action"].(string),
						User:      log["user_id"].(string),
					}

					// Handle changes array
					if changesArray, ok := log["changes"].(primitive.A); ok {
						for _, changeInterface := range changesArray {
							if change, ok := changeInterface.(bson.M); ok {
								entry.Changes = append(entry.Changes, domain.Change{
									Field:    change["field"].(string),
									OldValue: change["old_value"],
									NewValue: change["new_value"],
								})
							}
						}
					}

					history.Logs = append(history.Logs, entry)
				}
			}
		}
	}

	// Sort logs by timestamp descending
	sort.Slice(history.Logs, func(i, j int) bool {
		return history.Logs[i].Timestamp.After(history.Logs[j].Timestamp)
	})

	return &history, nil
}

func (r *ArchiveRepository) GetByCategory(ctx context.Context, category string, page, limit int) ([]domain.Archive, int64, error) {
	// Build filter for metadata.category
	filter := bson.M{
		"metadata.category": category,
		"$or": []bson.M{
			{"metadata.deleted_at": nil},
			{"metadata.deleted_at": bson.M{"$exists": false}},
		},
	}

	// Count total documents
	total, err := r.bucket.GetFilesCollection().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count documents: %v", err)
	}

	// Query with pagination
	opts := options.Find().
		SetSkip(int64((page - 1) * limit)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "metadata.updated_at", Value: -1}})

	cur, err := r.bucket.GetFilesCollection().Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find documents: %v", err)
	}
	defer cur.Close(ctx)

	var files []bson.M
	if err = cur.All(ctx, &files); err != nil {
		return nil, 0, fmt.Errorf("failed to decode documents: %v", err)
	}

	// Map results to Archive objects
	var archives []domain.Archive
	for _, file := range files {
		archive := mapToArchive(file)
		archives = append(archives, archive)
	}

	return archives, total, nil
}

func (r *ArchiveRepository) GetByTags(ctx context.Context, tags []string, page, limit int) ([]domain.Archive, int64, error) {
	// Build filter for metadata.tags
	filter := bson.M{
		"metadata.tags": bson.M{"$all": tags},
		"$or": []bson.M{
			{"metadata.deleted_at": nil},
			{"metadata.deleted_at": bson.M{"$exists": false}},
		},
	}

	// Count total documents
	total, err := r.bucket.GetFilesCollection().CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count documents: %v", err)
	}

	// Query with pagination
	opts := options.Find().
		SetSkip(int64((page - 1) * limit)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: "metadata.updated_at", Value: -1}})

	cur, err := r.bucket.GetFilesCollection().Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find documents: %v", err)
	}
	defer cur.Close(ctx)

	var files []bson.M
	if err = cur.All(ctx, &files); err != nil {
		return nil, 0, fmt.Errorf("failed to decode documents: %v", err)
	}

	// Map results to Archive objects
	var archives []domain.Archive
	for _, file := range files {
		archive := mapToArchive(file)
		archives = append(archives, archive)
	}

	return archives, total, nil
}

func (r *ArchiveRepository) CountDocuments(ctx context.Context, filter bson.M) (int64, error) {
	return r.bucket.GetFilesCollection().CountDocuments(ctx, filter)
}

func (r *ArchiveRepository) DeleteArchive(ctx context.Context, id string, permanent bool, userID string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid object ID: %v", err)
	}

	// Find the existing file first
	var result bson.M
	err = r.bucket.GetFilesCollection().FindOne(ctx, bson.M{"_id": objID}).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return domain.ErrArchiveNotFound
		}
		return fmt.Errorf("failed to find document: %v", err)
	}

	now := time.Now()
	metadata := result["metadata"].(bson.M)

	if permanent {
		// Permanent delete - remove file completely
		if err := r.bucket.Delete(objID); err != nil {
			return fmt.Errorf("failed to delete file: %v", err)
		}
		return nil
	}

	// Soft delete - update metadata
	changeLog := domain.ChangeLog{
		Timestamp: now,
		Action:    "delete",
		UserID:    userID,
		Changes: []domain.Change{
			{
				Field:    "status",
				OldValue: "active",
				NewValue: "deleted",
			},
		},
	}

	// Get existing change logs
	var changeLogs []domain.ChangeLog
	if logs, ok := metadata["change_logs"].(primitive.A); ok {
		for _, l := range logs {
			if log, ok := l.(bson.M); ok {
				changeLogs = append(changeLogs, domain.ChangeLog{
					Timestamp: log["timestamp"].(primitive.DateTime).Time(),
					Action:    log["action"].(string),
					UserID:    log["user_id"].(string),
				})
			}
		}
	}
	changeLogs = append(changeLogs, changeLog)

	// Update metadata
	update := bson.M{
		"$set": bson.M{
			"metadata.deleted_at":  now,
			"metadata.change_logs": changeLogs,
			"metadata.deleted_by":  userID,
			"metadata.updated_at":  now,
		},
	}

	_, err = r.bucket.GetFilesCollection().UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		return fmt.Errorf("failed to update document: %v", err)
	}

	return nil
}
