package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/Saaghh/lamoda-hr/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
	"time"
)

func (p *Postgres) TruncateTables(ctx context.Context) error {
	_, err := p.db.Exec(
		ctx,
		"TRUNCATE TABLE product_movements CASCADE")
	if err != nil {
		return fmt.Errorf("p.db.Exec(...): %w", err)
	}

	_, err = p.db.Exec(
		ctx,
		"TRUNCATE TABLE reservations CASCADE")
	if err != nil {
		return fmt.Errorf("p.db.Exec(...): %w", err)
	}

	_, err = p.db.Exec(
		ctx,
		"TRUNCATE TABLE stocks CASCADE")
	if err != nil {
		return fmt.Errorf("p.db.Exec(...): %w", err)
	}

	_, err = p.db.Exec(
		ctx,
		"TRUNCATE TABLE products CASCADE")
	if err != nil {
		return fmt.Errorf("p.db.Exec(...): %w", err)
	}

	_, err = p.db.Exec(
		ctx,
		"TRUNCATE TABLE warehouses CASCADE")
	if err != nil {
		return fmt.Errorf("p.db.Exec(...): %w", err)
	}

	return nil
}

func (p *Postgres) CreateWarehouse(ctx context.Context, warehouse model.Warehouse) (*model.Warehouse, error) {
	query := `
	INSERT INTO warehouses (id, name, is_active) 
	VALUES ($1, $2, $3)
	RETURNING created_at`

	err := p.db.QueryRow(
		ctx,
		query,
		warehouse.ID,
		warehouse.Name,
		warehouse.IsActive,
	).Scan(
		&warehouse.CreatedAt,
	)

	var pgErr *pgconn.PgError

	switch {
	case errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation:
		return nil, model.ErrObjectAlreadyExists
	case err != nil:
		return nil, fmt.Errorf("p.db.QueryRow(%s): %w", query, err)
	}

	return &warehouse, nil
}

func (p *Postgres) CreateProduct(ctx context.Context, product model.Product) (*model.Product, error) {
	query := `
	INSERT INTO products (sku, size, name)
	VALUES ($1, $2, $3)
	RETURNING created_at`

	err := p.db.QueryRow(
		ctx,
		query,
		product.SKU,
		product.Size,
		product.Name,
	).Scan(
		&product.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("p.db.QueryRow(%s): %w", query, err)
	}

	return &product, nil
}

func (p *Postgres) CreateStock(ctx context.Context, stock model.Stock) (*model.Stock, error) {
	query := `
	INSERT INTO stocks (warehouse_id, product_id, quantity) 
	VALUES ($1, $2, $3)
	RETURNING created_at, modified_at`

	err := p.db.QueryRow(
		ctx,
		query,
		stock.WarehouseID,
		stock.ProductID,
		stock.Quantity,
	).Scan(
		&stock.CreatedAt,
		&stock.ModifiedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("p.db.QueryRow(%s): %w", query, err)
	}

	return &stock, nil
}

func (p *Postgres) CreateReservations(ctx context.Context, reservations []model.Reservation) (*[]model.Reservation, error) {
	tx, err := p.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, fmt.Errorf("p.db.Begin(ctx): %w", err)
	}

	defer func() {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			zap.L().With(zap.Error(err)).Warn("CreateReservations/tx.Rollback(ctx)")
		}
	}()

	for i, value := range reservations {
		freeStock, err := GetFreeStock(ctx, tx, model.Stock{
			WarehouseID: value.WarehouseID,
			ProductID:   value.ProductID,
		})

		if err != nil {
			return nil, fmt.Errorf("GetFreeStock(...): %w", err)
		}

		if value.Quantity > freeStock.Quantity {
			return nil, model.ErrNotEnoughQuantity{
				SKU:              value.ProductID,
				RequiredQuantity: value.Quantity,
				ActualQuantity:   freeStock.Quantity,
			}
		}

		query := `
		INSERT INTO reservations (id, warehouse_id, product_id, quantity, created_at, due_date) 
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, warehouse_id, product_id, quantity, created_at, due_date`

		err = p.db.QueryRow(
			ctx,
			query,
			value.ID,
			value.WarehouseID,
			value.ProductID,
			value.Quantity,
			value.CreatedAt,
			value.DueDate,
		).Scan(
			&reservations[i].ID,
			&reservations[i].WarehouseID,
			&reservations[i].ProductID,
			&reservations[i].Quantity,
			&reservations[i].CreatedAt,
			&reservations[i].DueDate,
		)

		if err != nil {
			return nil, fmt.Errorf("p.db.QueryRow(%s): %w", query, err)
		}

	}

	time.Sleep(time.Second)

	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("tx.Commit(ctx): %w", err)
	}

	return &reservations, nil
}

func GetFreeStock(ctx context.Context, tx pgx.Tx, stock model.Stock) (*model.Stock, error) {
	//query := `
	//	SELECT quantity
	//	FROM stocks WHERE warehouse_id = $1 AND product_id = $2`

	query := `UPDATE stocks SET modified_at = now()
	         WHERE warehouse_id = $1 AND product_id = $2
	         RETURNING quantity`

	err := tx.QueryRow(
		ctx,
		query,
		stock.WarehouseID,
		stock.ProductID,
	).Scan(
		&stock.Quantity,
	)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, model.ErrStockNotFound{
			SKU:         stock.ProductID,
			WarehouseID: stock.WarehouseID,
		}
	case err != nil:
		return nil, fmt.Errorf("tx.QueryRow(%s): %w", query, err)
	}

	query = `
		SELECT quantity
		FROM reservations WHERE product_id = $1 AND warehouse_id = $2 AND is_active = true AND due_date < NOW()`

	rows, err := tx.Query(
		ctx,
		query,
		stock.ProductID,
		stock.WarehouseID)

	fmt.Println("")

	switch {
	case err != nil:
		return nil, fmt.Errorf("tx.Query(%s): %w", query, err)
	default:
		defer rows.Close()
		for rows.Next() {
			var reservedQuantity uint = 0

			err = rows.Scan(&reservedQuantity)
			if err != nil {
				return nil, fmt.Errorf("rows.Scan(&reservedQuantity): %w", err)
			}

			stock.Quantity -= reservedQuantity
		}
	}

	return &stock, nil
}

func (p *Postgres) DeleteReservations(ctx context.Context, reservations []model.Reservation) error {
	panic("implement me")
}

func (p *Postgres) GetWarehouseStocks(ctx context.Context, warehouseID uuid.UUID) (*[]model.Stock, error) {
	//TODO implement me
	panic("implement me")
}

func (p *Postgres) GetStocks(ctx context.Context) (*[]model.Stock, error) {
	//TODO implement me
	panic("implement me")
}
