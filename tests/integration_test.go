package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/Saaghh/lamoda-hr/internal/apiserver"
	"github.com/Saaghh/lamoda-hr/internal/config"
	"github.com/Saaghh/lamoda-hr/internal/logger"
	"github.com/Saaghh/lamoda-hr/internal/model"
	"github.com/Saaghh/lamoda-hr/internal/service"
	"github.com/Saaghh/lamoda-hr/internal/store"
	"github.com/google/uuid"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

const (
	bindAddr                   = "http://localhost:8081/api/v1"
	createReservationsEndpoint = "/createReservations"
	deleteReservationsEndpoint = "/deleteReservations"
	getStocksEndpoint          = "/getStocks"
)

type IntegrationTestSuite struct {
	suite.Suite
	str *store.Postgres

	ctx context.Context

	warehouses   []model.Warehouse
	products     []model.Product
	stocks       []model.Stock
	reservations []model.Reservation
}

func (s *IntegrationTestSuite) TearDownSuite() {
	for _, value := range s.reservations {
		err := s.str.DeleteRow(context.Background(), value)
		s.Require().NoError(err)
	}

	for _, value := range s.stocks {
		err := s.str.DeleteRow(context.Background(), value)
		s.Require().NoError(err)
	}

	for _, value := range s.products {
		err := s.str.DeleteRow(context.Background(), value)
		s.Require().NoError(err)
	}

	for _, value := range s.warehouses {
		err := s.str.DeleteRow(context.Background(), value)
		s.Require().NoError(err)
	}
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	s.ctx = ctx

	cfg := config.New()

	logger.InitLogger(logger.Config{Level: "warn"})

	// no error handling for now
	// check https://github.com/uber-go/zap/issues/991
	//nolint: errcheck
	defer zap.L().Sync()

	pgStore, err := store.New(ctx, cfg)
	if err != nil {
		zap.L().With(zap.Error(err)).Panic("main/pgStore.New(ctx, cfg)")
	}

	if err = pgStore.Migrate(migrate.Up); err != nil {
		zap.L().With(zap.Error(err)).Panic("main/pgStore.Migrate(migrate.Up)")
	}

	zap.L().Info("successful migration")

	s.str = pgStore

	s.createTestData()

	serviceLayer := service.New(pgStore)

	server := apiserver.New(apiserver.Config{BindAddress: ":8081"}, serviceLayer)

	go func() {
		err := server.Run(ctx)
		s.Require().NoError(err)
	}()

	go func() {
		err := serviceLayer.RunReservationsDeactivations(ctx, time.Second/10)
		s.Require().NoError(err)
	}()
}

func (s *IntegrationTestSuite) createTestData() {
	s.T().Helper()

	// create warehouses
	for i := 0; i < 3; i++ {
		wh, err := s.str.CreateWarehouse(s.ctx, model.Warehouse{
			ID:       uuid.New(),
			Name:     "warehouse #" + strconv.Itoa(i),
			IsActive: true,
		})

		s.Require().NoError(err)

		s.warehouses = append(s.warehouses, *wh)
	}

	// create products
	for i := 0; i < 3; i++ {
		pd, err := s.str.CreateProduct(s.ctx, model.Product{
			Name: "Футболка #" + strconv.Itoa(i*i),
			Size: "",
			SKU:  "product" + strconv.Itoa(i),
		})

		s.Require().NoError(err)

		s.products = append(s.products, *pd)
	}

	// create stocks
	for _, wh := range s.warehouses {
		for _, pd := range s.products {
			stock, err := s.str.CreateStock(s.ctx, model.Stock{
				WarehouseID: wh.ID,
				ProductID:   pd.SKU,
				Quantity:    100,
			})

			s.Require().NoError(err)

			s.stocks = append(s.stocks, *stock)
		}
	}
}

func (s *IntegrationTestSuite) TestReservations() {
	s.Run("POST:/createReservations", func() {
		s.Run("201", func() {
			requestReservations := []model.Reservation{
				{
					ID:          uuid.New(),
					WarehouseID: s.warehouses[0].ID,
					ProductID:   s.products[0].SKU,
					Quantity:    50,
					DueDate:     time.Now().Add(time.Hour * 24 * 30),
				},
				{
					ID:          uuid.New(),
					WarehouseID: s.warehouses[0].ID,
					ProductID:   s.products[1].SKU,
					Quantity:    50,
					DueDate:     time.Now().Add(time.Hour * 24 * 30),
				},
				{
					ID:          uuid.New(),
					WarehouseID: s.warehouses[0].ID,
					ProductID:   s.products[2].SKU,
					Quantity:    50,
					DueDate:     time.Now().Add(time.Hour * 24 * 30),
				},
			}
			s.reservations = make([]model.Reservation, 0)

			resp := s.sendRequest(
				context.Background(),
				http.MethodPost,
				createReservationsEndpoint,
				requestReservations,
				&apiserver.HTTPResponse{Data: &s.reservations})

			s.Require().Equal(http.StatusCreated, resp.StatusCode)
			s.Require().Equal(3, len(s.reservations))

			s.Run("429", func() {
				resp = s.sendRequest(
					context.Background(),
					http.MethodPost,
					createReservationsEndpoint,
					requestReservations,
					nil)

				s.Require().Equal(http.StatusTooManyRequests, resp.StatusCode)
			})
		})

		s.Run("422/overbooking", func() {
			requestReservations := []model.Reservation{
				{
					ID:          uuid.New(),
					WarehouseID: s.warehouses[0].ID,
					ProductID:   s.products[0].SKU,
					Quantity:    90,
					DueDate:     time.Now().Add(time.Hour * 24 * 30),
				},
			}

			resp := s.sendRequest(
				context.Background(),
				http.MethodPost,
				createReservationsEndpoint,
				requestReservations,
				nil)

			s.Require().Equal(http.StatusUnprocessableEntity, resp.StatusCode)
		})

		s.Run("404", func() {
			requestReservations := []model.Reservation{
				{
					ID:          uuid.New(),
					WarehouseID: uuid.New(),
					ProductID:   s.products[0].SKU,
					Quantity:    1,
					DueDate:     time.Now().Add(time.Hour * 24 * 30),
				},
			}

			resp := s.sendRequest(
				context.Background(),
				http.MethodPost,
				createReservationsEndpoint,
				requestReservations,
				nil)

			s.Require().Equal(http.StatusNotFound, resp.StatusCode)
		})

		s.Run("400", func() {
			s.Run("invalidUUID", func() {
				requestReservations := []model.Reservation{
					{
						ID:          uuid.Nil,
						WarehouseID: uuid.Nil,
						ProductID:   s.products[0].SKU,
						Quantity:    1,
						DueDate:     time.Now().Add(time.Hour * 24 * 30),
					},
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPost,
					createReservationsEndpoint,
					requestReservations,
					nil)

				s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
			})

			s.Run("invalidSKU", func() {
				requestReservations := []model.Reservation{
					{
						ID:          uuid.New(),
						WarehouseID: s.warehouses[0].ID,
						ProductID:   s.products[0].SKU + s.products[0].SKU,
						Quantity:    90,
						DueDate:     time.Now().Add(time.Hour * 24 * 30),
					},
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPost,
					createReservationsEndpoint,
					requestReservations,
					nil)

				s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
			})

			s.Run("invalidQuantity", func() {
				requestReservations := []model.Reservation{
					{
						ID:          uuid.New(),
						WarehouseID: s.warehouses[0].ID,
						ProductID:   s.products[0].SKU,
						Quantity:    0,
						DueDate:     time.Now().Add(time.Hour * 24 * 30),
					},
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPost,
					createReservationsEndpoint,
					requestReservations,
					nil)

				s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
			})

			s.Run("invalidQuantity", func() {
				requestReservations := []model.Reservation{
					{
						ID:          uuid.New(),
						WarehouseID: s.warehouses[0].ID,
						ProductID:   s.products[0].SKU,
						Quantity:    10,
						DueDate:     time.Now(),
					},
				}

				resp := s.sendRequest(
					context.Background(),
					http.MethodPost,
					createReservationsEndpoint,
					requestReservations,
					nil)

				s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
			})
		})

		s.Run("200/outdated transaction", func() {
			requestReservations := []model.Reservation{
				{
					ID:          uuid.New(),
					WarehouseID: s.warehouses[0].ID,
					ProductID:   s.products[0].SKU,
					Quantity:    50,
					DueDate:     time.Now().Add(time.Second / 10),
				},
			}

			var reservations []model.Reservation

			resp := s.sendRequest(
				context.Background(),
				http.MethodPost,
				createReservationsEndpoint,
				requestReservations,
				&apiserver.HTTPResponse{Data: &reservations})

			s.Require().Equal(http.StatusCreated, resp.StatusCode)

			for _, value := range reservations {
				s.reservations = append(s.reservations, value)
			}

			time.Sleep(time.Second / 2)

			requestReservations = []model.Reservation{
				{
					ID:          uuid.New(),
					WarehouseID: s.warehouses[0].ID,
					ProductID:   s.products[0].SKU,
					Quantity:    50,
					DueDate:     time.Now().Add(time.Hour * 24 * 30),
				},
			}

			resp = s.sendRequest(
				context.Background(),
				http.MethodPost,
				createReservationsEndpoint,
				requestReservations,
				&apiserver.HTTPResponse{Data: &reservations})

			for _, value := range reservations {
				s.reservations = append(s.reservations, value)
			}

			s.Require().Equal(http.StatusCreated, resp.StatusCode)
		})
	})

	s.Run("POST:/deleteReservations", func() {
		s.Run("204", func() {
			resp := s.sendRequest(
				context.Background(),
				http.MethodPost,
				deleteReservationsEndpoint,
				s.reservations[:3],
				nil,
			)

			s.Require().Equal(http.StatusNoContent, resp.StatusCode)
		})

		s.Run("400/invalidUUID", func() {
			requestReservations := []model.Reservation{
				{
					ID: uuid.Nil,
				},
			}

			resp := s.sendRequest(
				context.Background(),
				http.MethodPost,
				deleteReservationsEndpoint,
				requestReservations,
				nil,
			)

			s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
		})

		s.Run("404", func() {
			requestReservations := []model.Reservation{
				{
					ID: uuid.New(),
				},
			}

			resp := s.sendRequest(
				context.Background(),
				http.MethodPost,
				deleteReservationsEndpoint,
				requestReservations,
				nil,
			)

			s.Require().Equal(http.StatusNotFound, resp.StatusCode)
		})
	})

	s.Run("GET:/getStocks", func() {
		s.Run("200/0-warehouse", func() {
			var stocks []model.Stock

			params := model.GetParams{
				Limit:           10,
				WarehouseFilter: s.warehouses[0].ID.String(),
			}

			resp := s.sendRequest(
				context.Background(),
				http.MethodPost,
				getStocksEndpoint,
				params,
				&apiserver.HTTPResponse{Data: &stocks})

			s.Require().Equal(http.StatusOK, resp.StatusCode)
			s.Require().Equal(3, len(stocks))
		})

		s.Run("200/with-limit", func() {
			var stocks []model.Stock

			params := model.GetParams{
				Limit:           1,
				WarehouseFilter: s.warehouses[0].ID.String(),
			}

			resp := s.sendRequest(
				context.Background(),
				http.MethodPost,
				getStocksEndpoint,
				params,
				&apiserver.HTTPResponse{Data: &stocks})

			s.Require().Equal(http.StatusOK, resp.StatusCode)
			s.Require().Equal(1, len(stocks))
			s.Require().Equal(uint(50), stocks[0].ReservedQuantity)
			s.Require().Equal(uint(100), stocks[0].Quantity)
		})

		s.Run("200/empty-result", func() {
			var stocks []model.Stock

			params := model.GetParams{
				Limit:           10,
				WarehouseFilter: uuid.New().String(),
			}

			resp := s.sendRequest(
				context.Background(),
				http.MethodPost,
				getStocksEndpoint,
				params,
				&apiserver.HTTPResponse{Data: &stocks})

			s.Require().Equal(http.StatusOK, resp.StatusCode)
			s.Require().Equal(0, len(stocks))
		})

		s.Run("200/sorted", func() {
			var stocks []model.Stock

			params := model.GetParams{
				Limit:           10,
				WarehouseFilter: s.warehouses[0].ID.String(),
				Sorting:         "reserved_quantity",
			}

			resp := s.sendRequest(
				context.Background(),
				http.MethodPost,
				getStocksEndpoint,
				params,
				&apiserver.HTTPResponse{Data: &stocks})

			s.Require().Equal(http.StatusOK, resp.StatusCode)
			s.Require().Equal(3, len(stocks))
			s.Require().Equal(uint(0), stocks[0].ReservedQuantity)
			s.Require().Equal(uint(0), stocks[1].ReservedQuantity)
			s.Require().Equal(uint(50), stocks[2].ReservedQuantity)
		})

		s.Run("400", func() {
			params := model.GetParams{
				WarehouseFilter: "this is a sql injection",
			}

			resp := s.sendRequest(
				context.Background(),
				http.MethodPost,
				getStocksEndpoint,
				params,
				nil)

			s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
		})

		s.Run("400", func() {
			params := model.GetParams{
				ProductFilter: "1231231231234",
			}

			resp := s.sendRequest(
				context.Background(),
				http.MethodPost,
				getStocksEndpoint,
				params,
				nil)

			s.Require().Equal(http.StatusBadRequest, resp.StatusCode)
		})
	})
}

func (s *IntegrationTestSuite) sendRequest(ctx context.Context, method, endpoint string, body interface{}, dest interface{}) *http.Response {
	s.T().Helper()

	reqBody, err := json.Marshal(body)
	s.Require().NoError(err)

	req, err := http.NewRequestWithContext(ctx, method, bindAddr+endpoint, bytes.NewReader(reqBody))
	s.Require().NoError(err)

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)

	// resultString, err := io.ReadAll(resp.Body)
	// s.Require().NoError(err)
	// s.Require().NotNil(resultString)

	defer func() {
		err = resp.Body.Close()
		s.Require().NoError(err)
	}()

	if dest != nil {
		err = json.NewDecoder(resp.Body).Decode(&dest)
		s.Require().NoError(err)
	}

	return resp
}
