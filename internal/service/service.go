package service

import (
	"context"
	"fmt"
	"time"

	"github.com/Saaghh/lamoda-hr/internal/model"
	"github.com/google/uuid"
)

type store interface {
	CreateReservations(ctx context.Context, reservations []model.Reservation) (*[]model.Reservation, error)
	DeleteReservations(ctx context.Context, reservations []model.Reservation) error
	GetStocks(ctx context.Context, params model.GetParams) (*[]model.Stock, error)

	DeactivateDueReservations(ctx context.Context) error
}

type Service struct {
	db store
}

func New(db store) *Service {
	return &Service{
		db: db,
	}
}

func (s *Service) CreateReservations(ctx context.Context, reservations []model.Reservation) (*[]model.Reservation, error) {
	for _, value := range reservations {
		if err := model.ValidateReservationRequest(value); err != nil {
			return nil, fmt.Errorf("model.ValidateReservationRequest(value): %w", err)
		}
	}

	result, err := s.db.CreateReservations(ctx, reservations)
	if err != nil {
		return nil, fmt.Errorf("s.db.CreateReservations(ctx, reservations): %w", err)
	}

	return result, nil
}

func (s *Service) DeleteReservations(ctx context.Context, reservations []model.Reservation) error {
	for _, value := range reservations {
		if value.ID == uuid.Nil {
			return model.ErrInvalidUUID
		}
	}

	err := s.db.DeleteReservations(ctx, reservations)
	if err != nil {
		return fmt.Errorf("s.db.DeleteReservations(ctx, reservations): %w", err)
	}

	return nil
}

func (s *Service) GetStocks(ctx context.Context, params model.GetParams) (*[]model.Stock, error) {
	if params.Limit == 0 {
		params.Limit = 10
	}

	if err := model.ValidateGetParams(params); err != nil {
		return nil, fmt.Errorf("m odel.ValidateGetParams(params): %w", err)
	}

	stocks, err := s.db.GetStocks(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("s.db.GetWarehouseStocks(ctx, warehouseID): %w", err)
	}

	return stocks, nil
}

func (s *Service) RunReservationsDeactivations(ctx context.Context, duration time.Duration) error {
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		if err := s.db.DeactivateDueReservations(ctx); err != nil {
			return fmt.Errorf("s.db.DeactivateDueReservations(ctx): %w", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
