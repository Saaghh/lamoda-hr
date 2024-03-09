package tests

import (
	"bytes"
	"context"
	"encoding/json"
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
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"testing"
	"time"
)

const (
	bindAddr             = "http://localhost:8080/api/v1"
	reservationsEndpoint = "/reservations"
)

type IntegrationTestSuite struct {
	suite.Suite
	str *store.Postgres

	ctx context.Context

	warehouses []model.Warehouse
	products   []model.Product
}

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) SetupSuite() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	s.ctx = ctx

	cfg := config.New()

	logger.InitLogger(logger.Config{Level: cfg.LogLevel})

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

	serviceLayer := service.New(pgStore)

	server := apiserver.New(
		apiserver.Config{BindAddress: cfg.BindAddress},
		serviceLayer,
	)

	go func() {
		err := server.Run(context.Background())
		s.Require().NoError(err)
	}()

	s.createTestData()
}

func (s *IntegrationTestSuite) TearDownSuite() {
	err := s.str.TruncateTables(context.Background())
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) createTestData() {
	s.T().Helper()

	//create warehouses
	for i := 0; i < 3; i++ {
		wh, err := s.str.CreateWarehouse(s.ctx, model.Warehouse{
			ID:       uuid.New(),
			Name:     "warehouse #" + strconv.Itoa(i),
			IsActive: true,
		})

		s.Require().NoError(err)

		s.warehouses = append(s.warehouses, *wh)
	}

	//create products
	for i := 0; i < 3; i++ {
		pd, err := s.str.CreateProduct(s.ctx, model.Product{
			Name: "Футболка #" + strconv.Itoa(i*i),
			Size: "",
			SKU:  "product" + strconv.Itoa(i),
		})

		s.Require().NoError(err)

		s.products = append(s.products, *pd)
	}

	//create stocks
	for _, wh := range s.warehouses {
		for _, pd := range s.products {
			_, err := s.str.CreateStock(s.ctx, model.Stock{
				WarehouseID: wh.ID,
				ProductID:   pd.SKU,
				Quantity:    100,
			})

			s.Require().NoError(err)
		}
	}
}

func (s *IntegrationTestSuite) TestReservations() {
	s.Run("POST:/reservations", func() {
		requestReservations := []model.Reservation{
			{
				ID:          uuid.New(),
				WarehouseID: s.warehouses[0].ID,
				ProductID:   s.products[0].SKU,
				Quantity:    90,
				DueDate:     time.Now().Add(time.Hour * 24 * 30),
			},
		}
		resultReservations := make([]model.Reservation, 0)

		resp := s.sendRequest(
			context.Background(),
			http.MethodPost,
			reservationsEndpoint,
			requestReservations,
			&apiserver.HTTPResponse{Data: &resultReservations})

		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().Equal(3, len(resultReservations))
	})

	s.Run("POST:/reservations/2", func() {
		requestReservations := []model.Reservation{
			{
				ID:          uuid.New(),
				WarehouseID: s.warehouses[0].ID,
				ProductID:   s.products[0].SKU,
				Quantity:    90,
				DueDate:     time.Now().Add(time.Hour * 24 * 30),
			},
		}
		resultReservations := make([]model.Reservation, 0)

		resp := s.sendRequest(
			context.Background(),
			http.MethodPost,
			reservationsEndpoint,
			requestReservations,
			&apiserver.HTTPResponse{Data: &resultReservations})

		s.Require().Equal(http.StatusCreated, resp.StatusCode)
		s.Require().Equal(3, len(resultReservations))
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

	//resultString, err := io.ReadAll(resp.Body)
	//s.Require().NoError(err)
	//s.Require().NotNil(resultString)

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
