package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

const SKUMaxLength = 12

type Warehouse struct {
	ID        uuid.UUID
	Name      string
	IsActive  bool
	CreatedAt time.Time
}

type Product struct {
	Name      string    `json:"name,omitempty"`
	Size      string    `json:"size,omitempty"`
	SKU       string    `json:"sku"`
	CreatedAt time.Time `json:"createdAt"`
}

type Stock struct {
	WarehouseID      uuid.UUID `json:"warehouseId"`
	ProductID        string    `json:"productId"`
	Quantity         uint      `json:"quantity"`
	ReservedQuantity uint      `json:"reservedQuantity"`
	CreatedAt        time.Time `json:"createdAt,omitempty"`
	ModifiedAt       time.Time `json:"modifiedAt,omitempty"`
}

type Reservation struct {
	ID          uuid.UUID `json:"id"`
	WarehouseID uuid.UUID `json:"warehouseId"`
	ProductID   string    `json:"productId"`
	Quantity    uint      `json:"quantity"`
	CreatedAt   time.Time `json:"createdAt"`
	DueDate     time.Time `json:"dueDate"`
}

type GetParams struct {
	Offset          uint   `json:"offset,omitempty"`
	Limit           uint   `json:"limit,omitempty"`
	Sorting         string `json:"sorting,omitempty"`
	Descending      bool   `json:"descending,omitempty"`
	WarehouseFilter string `json:"warehouseFilter,omitempty"`
	ProductFilter   string `json:"productFilter,omitempty"`
}

func ValidateReservationRequest(reservation Reservation) error {
	if reservation.ID == uuid.Nil {
		return ErrInvalidUUID
	}

	if err := uuid.Validate(reservation.WarehouseID.String()); err != nil {
		return ErrInvalidUUID
	}

	if len(reservation.ProductID) > SKUMaxLength || reservation.ProductID == "" {
		return ErrInvalidSKU
	}

	if time.Now().After(reservation.DueDate) {
		return ErrIncorrectDueDate
	}

	if reservation.Quantity == 0 {
		return ErrInvalidQuantity
	}

	return nil
}

func ValidateGetParams(params GetParams) error {
	words := strings.Fields(params.Sorting)
	if len(words) > 1 {
		return ErrInvalidGetParams
	}

	words = strings.Fields(params.WarehouseFilter)
	if len(words) > 1 {
		return ErrInvalidGetParams
	}

	words = strings.Fields(params.ProductFilter)
	if len(words) > 1 {
		return ErrInvalidGetParams
	}

	if len(params.ProductFilter) > SKUMaxLength {
		return ErrInvalidSKU
	}

	return nil
}
