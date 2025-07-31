package application

import (
	"context"
	"time"

	"github.com/yhartanto178dev/archiven-api/internal/archive/domain"
	"go.mongodb.org/mongo-driver/bson"
)

type ArchiveService struct {
	repo domain.ArchiveRepository
}

func NewArchiveService(repo domain.ArchiveRepository) *ArchiveService {
	return &ArchiveService{repo: repo}
}

func (s *ArchiveService) UploadArchive(ctx context.Context, file domain.FileContent, metadata domain.ArchiveMetadata) (*domain.Archive, error) {
	// Validasi unik
	existing, err := s.repo.FindExistingArchive(ctx, domain.Archive{
		Name:     file.Name,
		Category: metadata.Category,
		Type:     metadata.Type,
		Tags:     metadata.Tags,
		OwnerID:  metadata.OwnerID,
	})
	if err != nil {
		return nil, err
	}

	archive := domain.Archive{
		Name:        file.Name,
		Size:        file.Size,
		Category:    metadata.Category,
		Type:        metadata.Type,
		Tags:        metadata.Tags,
		Description: metadata.Description,
		OwnerID:     metadata.OwnerID,
	}

	// Jika sudah ada, update versi
	if existing != nil {
		archive.ID = existing.ID
		archive.Version = existing.Version + 1
		archive.CreatedAt = existing.CreatedAt
	}

	return s.repo.SaveWithVersioning(ctx, archive, file.Content)
}

func (s *ArchiveService) GetArchive(ctx context.Context, id string) (*domain.Archive, []byte, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *ArchiveService) ListArchives(ctx context.Context, page, limit int) ([]domain.Archive, int64, error) {
	return s.repo.FindAll(ctx, page, limit)
}

func (s *ArchiveService) GetArchivesByIDs(ctx context.Context, ids []string) ([]domain.Archive, error) {
	return s.repo.FindByIDs(ctx, ids)
}

func (s *ArchiveService) DeleteArchive(ctx context.Context, id string, deleteType domain.DeleteType) error {
	exists, err := s.repo.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return domain.ErrArchiveNotFound
	}

	return s.repo.Delete(ctx, id, deleteType)
}

func (s *ArchiveService) RestoreArchive(ctx context.Context, id string) error {
	return s.repo.RestoreArchive(ctx, id)
}

func (s *ArchiveService) CleanupExpiredFiles(ctx context.Context) (int64, error) {
	return s.repo.DeleteExpiredFiles(ctx)
}

func (s *ArchiveService) CleanupTempFiles(ctx context.Context) (int64, error) {
	filter := bson.M{
		"metadata.is_temp": true,
		"metadata.created_at": bson.M{
			"$lt": time.Now().Add(-24 * time.Hour),
		},
	}
	return s.repo.DeleteByFilter(ctx, filter)
}

func (s *ArchiveService) GetHistory(ctx context.Context, id string) (*domain.History, error) {
	return s.repo.GetHistory(ctx, id)
}

func (s *ArchiveService) GetByCategory(ctx context.Context, category string, page, limit int) ([]domain.Archive, int64, error) {
	if category == "" {
		return nil, 0, domain.ErrInvalidCategory
	}
	return s.repo.GetByCategory(ctx, category, page, limit)
}

func (s *ArchiveService) GetByTags(ctx context.Context, tags []string, page, limit int) ([]domain.Archive, int64, error) {
	if len(tags) == 0 {
		return nil, 0, domain.ErrTagsRequired
	}
	return s.repo.GetByTags(ctx, tags, page, limit)
}

func (s *ArchiveService) UpdateArchive(ctx context.Context, archive domain.Archive, content []byte) (*domain.Archive, error) {
	return s.repo.SaveWithVersioning(ctx, archive, content)
}
