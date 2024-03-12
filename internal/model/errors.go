package model

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	ErrObjectAlreadyExists = errors.New("err duplicate request")
	ErrNoRowsAffected      = errors.New("err no rows affected")
	ErrInvalidUUID         = errors.New("err invalid uuid")
	ErrInvalidSKU          = errors.New("err invalid sku")
	ErrIncorrectDueDate    = errors.New("err incorrect due date")
	ErrInvalidQuantity     = errors.New("err invalid quantity")
	ErrInvalidLimit        = errors.New("err invalid limit")
	ErrInvalidGetParams    = errors.New("err invalid get params")
)

type DuplicateReservationError struct {
	ReservationID uuid.UUID
}

func (e DuplicateReservationError) Error() string {
	return "err duplicate reservation of " + e.ReservationID.String()
}

type StockNotFoundError struct {
	SKU         string
	WarehouseID uuid.UUID
}

func (e StockNotFoundError) Error() string {
	return fmt.Sprintf("err stock of %s not found at %s", e.SKU, e.WarehouseID.String())
}

type NotEnoughQuantityError struct {
	SKU              string
	WarehouseID      uuid.UUID
	RequiredQuantity uint
}

func (e NotEnoughQuantityError) Error() string {
	return fmt.Sprintf("err quantity of %s less than %d at %s", e.SKU, e.RequiredQuantity, e.WarehouseID.String())
}

type ReservationNotFoundError struct {
	ReservationID uuid.UUID
}

func (e ReservationNotFoundError) Error() string {
	return fmt.Sprintf("err reservation %s not found", e.ReservationID.String())
}
