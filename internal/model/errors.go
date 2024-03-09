package model

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
)

var (
	ErrObjectAlreadyExists = errors.New("err duplicate request")
)

type ErrDuplicateReservation struct {
	ReservationID uuid.UUID
}

func (e ErrDuplicateReservation) Error() string {
	return fmt.Sprintf("err duplicate reservation of %s", e.ReservationID.String())
}

type ErrStockNotFound struct {
	SKU         string
	WarehouseID uuid.UUID
}

func (e ErrStockNotFound) Error() string {
	return fmt.Sprintf("err stock of %s not found at %s", e.SKU, e.WarehouseID.String())
}

type ErrNotEnoughQuantity struct {
	SKU              string
	RequiredQuantity uint
	ActualQuantity   uint
}

func (e ErrNotEnoughQuantity) Error() string {
	return fmt.Sprintf("err not enough quantity of %s. Required: %d; Found: %d", e.SKU, e.RequiredQuantity, e.ActualQuantity)
}
