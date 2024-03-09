package model

import (
	"time"

	"github.com/google/uuid"
)

type Warehouse struct {
	ID        uuid.UUID
	Name      string
	IsActive  bool
	CreatedAt time.Time
}

type Product struct {
	Name      string    `json:"name"`
	Size      string    `json:"size"`
	SKU       string    `json:"sku"`
	CreatedAt time.Time `json:"createdAt"`
}

type Stock struct {
	WarehouseID uuid.UUID `json:"warehouseId"`
	ProductID   string    `json:"productId"`
	Quantity    uint      `json:"quantity"`
	CreatedAt   time.Time `json:"createdAt"`
	ModifiedAt  time.Time `json:"modifiedAt"`
}

type Reservation struct {
	ID          uuid.UUID `json:"id"`
	WarehouseID uuid.UUID `json:"warehouseId"`
	ProductID   string    `json:"productId"`
	Quantity    uint      `json:"quantity"`
	CreatedAt   time.Time `json:"createdAt"`
	DueDate     time.Time `json:"dueDate"`
}

type ProductMovement struct {
	ID            uuid.UUID
	WarehouseID   uuid.UUID
	ProductID     uuid.UUID
	ReservationID uuid.UUID
	Quantity      uint
	CreatedAt     time.Time
}
