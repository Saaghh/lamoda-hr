package apiserver

import (
	"context"
	"encoding/json"
	"github.com/Saaghh/lamoda-hr/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"net/http"
)

type HTTPResponse struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

type service interface {
	CreateReservations(ctx context.Context, reservations []model.Reservation) (*[]model.Reservation, error)
	DeleteReservations(ctx context.Context, reservations []model.Reservation) error

	//TODO: pagination, filtration, sorting
	GetWarehouseStocks(ctx context.Context, warehouseID uuid.UUID) (*[]model.Stock, error)
	GetStocks(ctx context.Context) (*[]model.Stock, error)
}

func (s *APIServer) createReservations(w http.ResponseWriter, r *http.Request) {
	var reservations *[]model.Reservation

	if err := json.NewDecoder(r.Body).Decode(&reservations); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "error reading body")
	}

	reservations, err := s.service.CreateReservations(r.Context(), *reservations)

	switch {
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

	switch {
	case err != nil:
		zap.L().With(zap.Error(err)).Warn("deleteReservations/s.service.DeleteReservations(r.Context(), *reservations)")

		writeErrorResponse(w, http.StatusInternalServerError, "internal server error")

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *APIServer) getWarehouseStocks(w http.ResponseWriter, r *http.Request) {
	warehouseID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "error reading warehouse id")

		return
	}

	stocks, err := s.service.GetWarehouseStocks(r.Context(), warehouseID)

	switch {
	case err != nil:
		zap.L().With(zap.Error(err)).Warn("getWarehouseStocks/s.service.GetWarehouseStocks(r.Context(), warehouseID)")

		writeErrorResponse(w, http.StatusInternalServerError, "internal server error")

		return
	}

	writeOkResponse(w, http.StatusOK, stocks)
}

func (s *APIServer) getStocks(w http.ResponseWriter, r *http.Request) {
	stocks, err := s.service.GetStocks(r.Context())

	switch {
	case err != nil:
		zap.L().With(zap.Error(err)).Warn("getStocks/s.service.GetStocks(r.Context())")

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
