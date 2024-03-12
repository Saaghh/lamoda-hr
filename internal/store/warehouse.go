package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Saaghh/lamoda-hr/internal/model"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

func (p *Postgres) DeleteRow(ctx context.Context, object any) error {
	switch v := object.(type) {
	case model.Stock:
		query := `DELETE FROM stocks WHERE product_id = $1 and warehouse_id = $2`

		_, err := p.db.Exec(ctx, query, v.ProductID, v.WarehouseID)
		if err != nil {
			return fmt.Errorf("p.db.Exec(ctx, query, v.ProductID, v.WarehouseID): %w", err)
		}
	case model.Reservation:
		query := `DELETE FROM reservations WHERE id = $1`

		_, err := p.db.Exec(ctx, query, v.ID)
		if err != nil {
			return fmt.Errorf("p.db.Exec(ctx, query, v.ID): %w", err)
		}
	case model.Product:
		query := `DELETE FROM products WHERE sku = $1`

		_, err := p.db.Exec(ctx, query, v.SKU)
		if err != nil {
			return fmt.Errorf("p.db.Exec(ctx, query, v.SKU): %w", err)
		}
	case model.Warehouse:
		query := `DELETE FROM warehouses WHERE id = $1`

		_, err := p.db.Exec(ctx, query, v.ID)
		if err != nil {
			return fmt.Errorf("p.db.Exec(ctx, query, v.ID): %w", err)
		}
	default:
		return errors.ErrUnsupported
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
	INSERT INTO stocks (warehouse_id, product_id, quantity, reserved_quantity) 
	VALUES ($1, $2, $3, 0)
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
	tx, err := p.db.Begin(ctx)
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
		SET reserved_quantity = reserved_quantity + $1, modified_at = now() 
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
			return nil, &model.NotEnoughQuantityError{
				SKU:              value.ProductID,
				RequiredQuantity: value.Quantity,
				WarehouseID:      value.WarehouseID,
			}
		case errors.Is(err, pgx.ErrNoRows):
			return nil, &model.StockNotFoundError{
				SKU:         value.ProductID,
				WarehouseID: value.WarehouseID,
			}
		case err != nil:
			return nil, fmt.Errorf("tx.QueryRow(%s): %w", query, err)
		}

		query = `
		INSERT INTO reservations (id, warehouse_id, product_id, quantity, due_date) 
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, warehouse_id, product_id, quantity, created_at, due_date`

		err = tx.QueryRow(
			ctx,
			query,
			value.ID,
			value.WarehouseID,
			value.ProductID,
			value.Quantity,
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
			return nil, &model.DuplicateReservationError{ReservationID: value.ID}
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

//nolint:cyclop
func (p *Postgres) DeactivateDueReservations(ctx context.Context) error {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("p.db.Begin(ctx): %w", err)
	}

	defer func() {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			zap.L().With(zap.Error(err)).Warn("DeactivateDueReservations/tx.Rollback(ctx)")
		}
	}()

	query := `
	UPDATE reservations 
	SET is_active = false 
	WHERE due_date < now() AND is_active = true 
	RETURNING product_id, warehouse_id, quantity`

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

	for _, value := range releasedReservations {
		query = `
		UPDATE stocks 
		SET reserved_quantity = reserved_quantity - $1, modified_at = now() 
		WHERE product_id = $2 and warehouse_id = $3
		RETURNING modified_at, reserved_quantity`

		commandTag, err := tx.Exec(
			ctx,
			query,
			value.Quantity,
			value.ProductID,
			value.WarehouseID,
		)
		if err != nil {
			return fmt.Errorf("tx.Exec(%s): %w", query, err)
		}

		if commandTag.RowsAffected() != 1 {
			return fmt.Errorf("tx.Exec(%s): %w", query, model.ErrNoRowsAffected)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("tx.Commit(ctx): %w", err)
	}

	return nil
}

func (p *Postgres) DeleteReservations(ctx context.Context, reservations []model.Reservation) error {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("p.db.Begin(ctx): %w", err)
	}

	defer func() {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			zap.L().With(zap.Error(err)).Warn("DeleteReservations/tx.Rollback(ctx)")
		}
	}()

	for _, value := range reservations {
		reservation := value
		query := `
			UPDATE reservations 
			SET is_active = false 
			WHERE is_active = true AND id = $1
			RETURNING product_id, warehouse_id, quantity`

		err := tx.QueryRow(
			ctx,
			query,
			reservation.ID,
		).Scan(
			&reservation.ProductID,
			&reservation.WarehouseID,
			&reservation.Quantity,
		)

		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return &model.ReservationNotFoundError{ReservationID: reservation.ID}
		case err != nil:
			return fmt.Errorf("tx.QueryRow(%s): %w", query, err)
		}

		query = `
		UPDATE stocks 
		SET reserved_quantity = reserved_quantity - $1, modified_at = now() 
		WHERE warehouse_id = $2 AND product_id = $3
		RETURNING modified_at`

		commandTag, err := tx.Exec(
			ctx,
			query,
			reservation.Quantity,
			reservation.WarehouseID,
			reservation.ProductID,
		)
		if err != nil {
			return fmt.Errorf("tx.QueryRow(%s): %w", query, err)
		}

		if commandTag.RowsAffected() != 1 {
			return fmt.Errorf("\"tx.QueryRow(%s): %w", query, model.ErrNoRowsAffected)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("tx.Commit(ctx): %w", err)
	}

	return nil
}

func (p *Postgres) GetStocks(ctx context.Context, params model.GetParams) (*[]model.Stock, error) {
	query := `SELECT warehouse_id, product_id, quantity, reserved_quantity, created_at, modified_at FROM stocks `

	var conditions []string

	if params.WarehouseFilter != "" {
		conditions = append(conditions, fmt.Sprintf("warehouse_id = '%s'", params.WarehouseFilter))
	}

	if params.ProductFilter != "" {
		conditions = append(conditions, fmt.Sprintf("product_id = '%s'", params.ProductFilter))
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	if params.Sorting != "" {
		query += " ORDER BY " + params.Sorting
		if params.Descending {
			query += " DESC"
		}
	}

	query += fmt.Sprintf(" OFFSET %d LIMIT %d", params.Offset, params.Limit)

	rows, err := p.db.Query(
		ctx,
		query)
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
			&stock.ReservedQuantity,
			&stock.CreatedAt,
			&stock.ModifiedAt)
		if err != nil {
			return nil, fmt.Errorf("rows.Scan(%s): %w", query, err)
		}

		stocks = append(stocks, stock)
	}

	return &stocks, nil
}
