package model

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrObjectAlreadyExists = errors.New("err duplicate request")

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
	WarehouseID      uuid.UUID
	RequiredQuantity uint
}

func (e ErrNotEnoughQuantity) Error() string {
	return fmt.Sprintf("err quantity of %s less than %d at %s", e.SKU, e.RequiredQuantity, e.WarehouseID.String())
}

type ErrReservationNotFound struct {
	ReservationID uuid.UUID
}

func (e ErrReservationNotFound) Error() string {
	return fmt.Sprintf("err reservation %s not found", e.ReservationID.String())
}
