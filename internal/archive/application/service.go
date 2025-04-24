package application

import (
	"context"

	"github.com/yhartanto178dev/archiven-api/internal/archive/domain"
)

type ArchiveService struct {
	repo domain.ArchiveRepository
}

func NewArchiveService(repo domain.ArchiveRepository) *ArchiveService {
	return &ArchiveService{repo: repo}
}

func (s *ArchiveService) UploadArchive(ctx context.Context, file domain.FileContent) error {
	return s.repo.Save(ctx, file)
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
	return s.repo.Restore(ctx, id)
}

func (s *ArchiveService) CleanupTempFiles(ctx context.Context) error {
	return s.repo.DeleteExpiredTempFiles(ctx)
}
