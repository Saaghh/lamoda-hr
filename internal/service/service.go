package service

import (
	"context"
	"fmt"

	"github.com/Saaghh/lamoda-hr/internal/model"
	"github.com/google/uuid"
)

type store interface {
	CreateReservations(ctx context.Context, reservations []model.Reservation) (*[]model.Reservation, error)
	DeleteReservations(ctx context.Context, reservations []model.Reservation) error
	GetWarehouseStocks(ctx context.Context, warehouseID uuid.UUID) (*[]model.Stock, error)
	GetStocks(ctx context.Context) (*[]model.Stock, error)

	CreateWarehouse(ctx context.Context, warehouse model.Warehouse) (*model.Warehouse, error)
	CreateProduct(ctx context.Context, product model.Product) (*model.Product, error)
	CreateStock(ctx context.Context, stock model.Stock) (*model.Stock, error)
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
	// TODO add validation

	result, err := s.db.CreateReservations(ctx, reservations)
	if err != nil {
		return nil, fmt.Errorf("s.db.CreateReservations(ctx, reservations): %w", err)
	}

	return result, nil
}

func (s *Service) DeleteReservations(ctx context.Context, reservations []model.Reservation) error {
	err := s.db.DeleteReservations(ctx, reservations)
	if err != nil {
		return fmt.Errorf("s.db.DeleteReservations(ctx, reservations): %w", err)
	}

	return nil
}

func (s *Service) GetWarehouseStocks(ctx context.Context, warehouseID uuid.UUID) (*[]model.Stock, error) {
	stocks, err := s.db.GetWarehouseStocks(ctx, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("s.db.GetWarehouseStocks(ctx, warehouseID): %w", err)
	}

	return stocks, nil
}

func (s *Service) GetStocks(ctx context.Context) (*[]model.Stock, error) {
	// TODO implement me
	panic("implement me")
}
