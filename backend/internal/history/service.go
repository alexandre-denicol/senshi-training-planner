package history

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/alexandre/senshi-training-planner/backend/internal/auth"
)

type Service struct {
	store   Store
	newUUID func() (string, error)
	now     func() time.Time
}

func NewService(store Store) *Service {
	return &Service{
		store:   store,
		newUUID: auth.NewUUID,
		now:     func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) List(ctx context.Context, from string, to string) ([]ListItem, error) {
	fromDate, err := parseDate(from)
	if err != nil {
		return nil, ErrInvalidRequest
	}
	toDate, err := parseDate(to)
	if err != nil {
		return nil, ErrInvalidRequest
	}
	if fromDate.After(toDate) {
		return nil, ErrInvalidRequest
	}
	if int(toDate.Sub(fromDate).Hours()/24)+1 > MaxRangeDays {
		return nil, ErrInvalidRequest
	}

	return s.store.ListHistory(ctx, from, to)
}

func (s *Service) Get(ctx context.Context, id string) (Detail, error) {
	if !validID(id) {
		return Detail{}, ErrNotFound
	}

	detail, err := s.store.GetHistory(ctx, id)
	return mapDetailResult(detail, err)
}

func (s *Service) Complete(ctx context.Context, scheduleEntryID string, completedBy auth.PublicUser) (Detail, error) {
	if !validID(scheduleEntryID) {
		return Detail{}, ErrScheduleNotFound
	}
	if strings.TrimSpace(completedBy.ID) == "" || strings.TrimSpace(completedBy.Name) == "" {
		return Detail{}, ErrInvalidRequest
	}

	historyID, err := s.newUUID()
	if err != nil {
		return Detail{}, err
	}

	detail, err := s.store.CompleteScheduleEntry(ctx, historyID, scheduleEntryID, completedBy, s.now())
	return mapDetailResult(detail, err)
}

func mapDetailResult(detail Detail, err error) (Detail, error) {
	if errors.Is(err, ErrNotFound) {
		return Detail{}, ErrNotFound
	}
	if errors.Is(err, ErrScheduleNotFound) {
		return Detail{}, ErrScheduleNotFound
	}
	if errors.Is(err, ErrAlreadyCompleted) {
		return Detail{}, ErrAlreadyCompleted
	}
	if errors.Is(err, ErrSnapshotUnavailable) {
		return Detail{}, ErrSnapshotUnavailable
	}
	if err != nil {
		return Detail{}, err
	}

	return detail, nil
}

func parseDate(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, err
	}
	if parsed.Format("2006-01-02") != value {
		return time.Time{}, ErrInvalidRequest
	}

	return parsed, nil
}

func validID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for index, value := range id {
		switch index {
		case 8, 13, 18, 23:
			if value != '-' {
				return false
			}
		default:
			if (value < '0' || value > '9') &&
				(value < 'a' || value > 'f') &&
				(value < 'A' || value > 'F') {
				return false
			}
		}
	}

	return true
}
