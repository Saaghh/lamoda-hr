package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Saaghh/lamoda-hr/internal/model"
	"go.uber.org/zap"
)

type HTTPResponse struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

type service interface {
	CreateReservations(ctx context.Context, reservations []model.Reservation) (*[]model.Reservation, error)
	DeleteReservations(ctx context.Context, reservations []model.Reservation) error

	GetStocks(ctx context.Context, params model.GetParams) (*[]model.Stock, error)
}

func (s *APIServer) createReservations(w http.ResponseWriter, r *http.Request) {
	var reservations *[]model.Reservation

	if err := json.NewDecoder(r.Body).Decode(&reservations); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "error reading body")

		return
	}

	reservations, err := s.service.CreateReservations(r.Context(), *reservations)

	var (
		errDuplicateReservation *model.DuplicateReservationError
		errStockNotFound        *model.StockNotFoundError
		errNotEnoughQuantity    *model.NotEnoughQuantityError
	)

	switch {
	case errors.Is(err, model.ErrInvalidSKU):
		writeErrorResponse(w, http.StatusBadRequest, "invalid product sku")

		return
	case errors.Is(err, model.ErrInvalidUUID):
		writeErrorResponse(w, http.StatusBadRequest, "invalid uuid")

		return
	case errors.Is(err, model.ErrInvalidQuantity):
		writeErrorResponse(w, http.StatusBadRequest, "invalid quantity")

		return
	case errors.Is(err, model.ErrIncorrectDueDate):
		writeErrorResponse(w, http.StatusBadRequest, "incorrect due date")

		return
	case errors.As(err, &errDuplicateReservation):
		writeErrorResponse(w, http.StatusTooManyRequests, errDuplicateReservation.Error())

		return
	case errors.As(err, &errNotEnoughQuantity):
		writeErrorResponse(w, http.StatusUnprocessableEntity, errNotEnoughQuantity.Error())

		return
	case errors.As(err, &errStockNotFound):
		writeErrorResponse(w, http.StatusNotFound, errStockNotFound.Error())

		return
	case err != nil:
		zap.L().With(zap.Error(err)).Warn("createReservations/s.service.CreateReservations(r.Context(), *reservations)")

		writeErrorResponse(w, http.StatusInternalServerError, "internal server error")

		return
	}

	writeOkResponse(w, http.StatusCreated, reservations)
}

func (s *APIServer) deleteReservations(w http.ResponseWriter, r *http.Request) {
	var reservations *[]model.Reservation

	if err := json.NewDecoder(r.Body).Decode(&reservations); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "error reading body")
	}

	err := s.service.DeleteReservations(r.Context(), *reservations)

	var errReservationNotFound *model.ReservationNotFoundError

	switch {
	case errors.Is(err, model.ErrInvalidUUID):
		writeErrorResponse(w, http.StatusBadRequest, "invalid uuid")

		return
	case errors.As(err, &errReservationNotFound):
		writeErrorResponse(w, http.StatusNotFound, errReservationNotFound.Error())

		return
	case err != nil:
		zap.L().With(zap.Error(err)).Warn("deleteReservations/s.service.DeleteReservations(r.Context(), *reservations)")

		writeErrorResponse(w, http.StatusInternalServerError, "internal server error")

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *APIServer) getStocks(w http.ResponseWriter, r *http.Request) {
	var params model.GetParams

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "error reading body")

		return
	}

	stocks, err := s.service.GetStocks(r.Context(), params)

	switch {
	case errors.Is(err, model.ErrInvalidSKU):
		fallthrough
	case errors.Is(err, model.ErrInvalidGetParams):
		writeErrorResponse(w, http.StatusBadRequest, "invalid get params")

		return
	case err != nil:
		zap.L().With(zap.Error(err)).Warn("getStocks/s.service.GetWarehouseStocks(r.Context(), warehouseID)")

		writeErrorResponse(w, http.StatusInternalServerError, "internal server error")

		return
	}

	writeOkResponse(w, http.StatusOK, stocks)
}

func writeOkResponse(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	err := json.NewEncoder(w).Encode(HTTPResponse{Data: data})
	if err != nil {
		zap.L().With(zap.Error(err)).Warn(
			"writeOkResponse/json.NewEncoder(w).Encode(HTTPResponse{Data: data})")
	}
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	err := json.NewEncoder(w).Encode(HTTPResponse{Error: description})
	if err != nil {
		zap.L().With(zap.Error(err)).Warn(
			"writeErrorResponse/json.NewEncoder(w).Encode(HTTPResponse{Error: data})")
	}
}
