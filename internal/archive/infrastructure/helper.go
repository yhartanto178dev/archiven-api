package infrastructure

import (
	"time"

	"github.com/yhartanto178dev/archiven-api/internal/archive/domain"
)

func CreateChangeLog(action, userID string, old, new *domain.Archive) domain.ChangeLog {
	changes := []domain.Change{}

	if old != nil {
		// Deteksi perubahan
		if old.Name != new.Name {
			changes = append(changes, domain.Change{
				Field:    "name",
				OldValue: old.Name,
				NewValue: new.Name,
			})
		}
		// Tambahkan field lainnya yang perlu dilacak...
	}

	return domain.ChangeLog{
		Timestamp: time.Now(),
		Action:    action,
		UserID:    userID,
		Changes:   changes,
	}
}
