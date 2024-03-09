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
	if err := p.DeactivateDueReservations(ctx); err != nil {
		return nil, fmt.Errorf("p.DeactivateDueReservations(ctx): %w", err)
	}

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

		query := `
		UPDATE stocks 
		SET quantity = quantity - $1, modified_at = now() 
		WHERE warehouse_id = $2 AND product_id = $3
		RETURNING modified_at, quantity`

		var freeStock model.Stock

		err = tx.QueryRow(
			ctx,
			query,
			value.Quantity,
			value.WarehouseID,
			value.ProductID,
		).Scan(
			&freeStock.ModifiedAt,
			&freeStock.Quantity,
		)

		var pgErr *pgconn.PgError

		switch {
		case errors.As(err, &pgErr) && pgErr.Code == pgerrcode.CheckViolation:
			return nil, &model.ErrNotEnoughQuantity{
				SKU:              value.ProductID,
				RequiredQuantity: value.Quantity,
				WarehouseID:      value.WarehouseID,
			}
		case errors.Is(err, pgx.ErrNoRows):
			return nil, &model.ErrStockNotFound{
				SKU:         value.ProductID,
				WarehouseID: value.WarehouseID,
			}
		case err != nil:
			return nil, fmt.Errorf("tx.QueryRow(%s): %w", query, err)
		}

		query = `
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

		switch {
		case errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation:
			return nil, &model.ErrDuplicateReservation{ReservationID: value.ID}
		case err != nil:
			return nil, fmt.Errorf("p.db.QueryRow(%s): %w", query, err)

		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("tx.Commit(ctx): %w", err)
	}

	return &reservations, nil
}

func (p *Postgres) DeactivateDueReservations(ctx context.Context) error {
	tx, err := p.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return fmt.Errorf("p.db.Begin(ctx): %w", err)
	}

	defer func() {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			zap.L().With(zap.Error(err)).Warn("CreateReservations/tx.Rollback(ctx)")
		}
	}()

	query := `
	UPDATE reservations 
	SET is_active = false 
	WHERE due_date < now() AND is_active = true 
	RETURNING product_id, warehouse_id, quantity`

	func() {
	}()
	rows, err := tx.Query(
		ctx,
		query)
	if err != nil {
		return fmt.Errorf("tx.Query(%s): %w", query, err)
	}

	defer rows.Close()

	releasedReservations := make([]model.Reservation, 0)

	for rows.Next() {
		var releasedReservation model.Reservation

		err = rows.Scan(
			&releasedReservation.ProductID,
			&releasedReservation.WarehouseID,
			&releasedReservation.Quantity,
		)
		if err != nil {
			return fmt.Errorf("rows.Scan(&reservationQuantity): %w", err)
		}

		releasedReservations = append(releasedReservations, releasedReservation)
	}

	if len(releasedReservations) == 0 {
		return nil
	}

	rows.Close()

	for _, value := range releasedReservations {
		query = `
		UPDATE stocks 
		SET quantity = quantity + $1, modified_at = now() 
		WHERE product_id = $2 and warehouse_id = $3
		RETURNING modified_at, quantity`

		err = tx.QueryRow(
			ctx,
			query,
			value.Quantity,
			value.ProductID,
			value.WarehouseID,
		).Scan(nil, nil)
		if err != nil {
			return fmt.Errorf("tx.QueryRow(%s): %w", query, err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("tx.Commit(ctx): %w", err)
	}

	return nil
}

func (p *Postgres) DeleteReservations(ctx context.Context, reservations []model.Reservation) error {
	tx, err := p.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return fmt.Errorf("p.db.Begin(ctx): %w", err)
	}

	defer func() {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			zap.L().With(zap.Error(err)).Warn("CreateReservations/tx.Rollback(ctx)")
		}
	}()

	for _, value := range reservations {
		query := `
			UPDATE reservations 
			SET is_active = false 
			WHERE is_active = true AND id = $1 
			RETURNING product_id, warehouse_id, quantity`

		err := tx.QueryRow(
			ctx,
			query,
			value.ID,
		).Scan(
			&value.ProductID,
			&value.WarehouseID,
			&value.Quantity,
		)

		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return &model.ErrReservationNotFound{ReservationID: value.ID}
		case err != nil:
			return fmt.Errorf("tx.QueryRow(%s): %w", query, err)
		}

		query = `
		UPDATE stocks 
		SET quantity = quantity + $1, modified_at = now() 
		WHERE warehouse_id = $2 AND product_id = $3
		RETURNING modified_at`

		err = tx.QueryRow(
			ctx,
			query,
			value.Quantity,
			value.WarehouseID,
			value.ProductID,
		).Scan(nil)
		if err != nil {
			return fmt.Errorf("tx.QueryRow(...): %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("tx.Commit(ctx): %w", err)
	}

	return nil
}

func (p *Postgres) GetWarehouseStocks(ctx context.Context, warehouseID uuid.UUID) (*[]model.Stock, error) {
	err := p.DeactivateDueReservations(ctx)
	if err != nil {
		return nil, fmt.Errorf("p.DeactivateDueReservations(ctx): %w", err)
	}

	query := `SELECT warehouse_id, product_id, quantity, created_at, modified_at FROM stocks WHERE warehouse_id = $1`

	rows, err := p.db.Query(
		ctx,
		query,
		warehouseID)
	if err != nil {
		return nil, fmt.Errorf("p.db.Query(%s): %w", query, err)
	}

	defer rows.Close()

	stocks := make([]model.Stock, 0)

	for rows.Next() {
		var stock model.Stock

		err = rows.Scan(
			&stock.WarehouseID,
			&stock.ProductID,
			&stock.Quantity,
			&stock.CreatedAt,
			&stock.ModifiedAt)
		if err != nil {
			return nil, fmt.Errorf("rows.Scan(...): %w", err)
		}

		stocks = append(stocks, stock)
	}

	return &stocks, nil
}

func (p *Postgres) GetStocks(ctx context.Context) (*[]model.Stock, error) {
	// TODO implement me
	panic("implement me")
}
