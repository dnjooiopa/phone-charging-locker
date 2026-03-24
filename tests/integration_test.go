package tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	_ "modernc.org/sqlite"

	"github.com/dnjooiopa/phone-charging-locker/internal/repository/locker_repository"
	"github.com/dnjooiopa/phone-charging-locker/internal/repository/session_repository"
	"github.com/dnjooiopa/phone-charging-locker/internal/server/gin_server"
	"github.com/dnjooiopa/phone-charging-locker/internal/usecase"
	"github.com/dnjooiopa/phone-charging-locker/schema"
)

// MockInvoiceRepository is a mock implementation of usecase.InvoiceRepository for integration tests
type MockInvoiceRepository struct {
	mock.Mock
}

func (m *MockInvoiceRepository) CreateInvoice(ctx context.Context, params *usecase.CreateInvoiceParams) (*usecase.CreateInvoiceResult, error) {
	args := m.Called(ctx, params)
	var r0 *usecase.CreateInvoiceResult
	if args.Get(0) != nil {
		r0 = args.Get(0).(*usecase.CreateInvoiceResult)
	}
	return r0, args.Error(1)
}

func (m *MockInvoiceRepository) RegisterWebhookEndpoint(ctx context.Context, webhookURL string) error {
	args := m.Called(ctx, webhookURL)
	return args.Error(0)
}

type IntegrationTestSuite struct {
	suite.Suite
	db     *sql.DB
	server *gin_server.Server
	dbPath string
}

// SetupSuite runs once before all tests in the suite
func (s *IntegrationTestSuite) SetupSuite() {
	ctx := context.Background()

	// Create temp SQLite database
	tmpDir, err := os.MkdirTemp("", "pcl-test-*")
	require.NoError(s.T(), err)
	s.dbPath = filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("sqlite", s.dbPath)
	require.NoError(s.T(), err)

	_, err = db.Exec("PRAGMA journal_mode=WAL")
	require.NoError(s.T(), err)
	_, err = db.Exec("PRAGMA foreign_keys=ON")
	require.NoError(s.T(), err)

	require.NoError(s.T(), db.Ping())
	s.db = db

	// Run schema migration
	err = schema.Migrate(ctx, db)
	require.NoError(s.T(), err)

	// Load seed data
	seedPath := filepath.Join(".", "seed.sql")
	seedSQL, err := os.ReadFile(seedPath)
	require.NoError(s.T(), err)
	_, err = db.Exec(string(seedSQL))
	require.NoError(s.T(), err)

	// Initialize server
	lockerRepository := locker_repository.New()
	sessionRepository := session_repository.New()

	invoiceRepo := &MockInvoiceRepository{}
	invoiceRepo.On("CreateInvoice", mock.Anything, mock.AnythingOfType("*usecase.CreateInvoiceParams")).Return(&usecase.CreateInvoiceResult{
		PaymentHash: "testhash123",
		Serialized:  "lntb1u1testinvoice",
	}, nil)

	uc := usecase.New(
		&usecase.Config{
			ChargingDuration: 1 * time.Hour,
			ChargingAmount:   2000,
		},
		lockerRepository,
		sessionRepository,
		invoiceRepo,
	)

	server := gin_server.New(&gin_server.Config{
		Environment: "test",
	}, uc)
	server.Use(gin_server.ErrorHandler())
	server.Use(gin_server.DatabaseMiddleware(db))
	server.SetUpRoutes()
	s.server = server
}

// TearDownSuite runs once after all tests in the suite
func (s *IntegrationTestSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
	if s.dbPath != "" {
		os.RemoveAll(filepath.Dir(s.dbPath))
	}
}

// Helper function to make HTTP requests
func (s *IntegrationTestSuite) makeRequest(method, path string, body interface{}) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		require.NoError(s.T(), err)
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.server.Handler().ServeHTTP(w, req)
	return w
}

// ============================================
// HEALTH CHECK TESTS
// ============================================

func (s *IntegrationTestSuite) TestHealthCheck() {
	resp := s.makeRequest("GET", "/healthz", nil)
	s.Equal(http.StatusOK, resp.Code)
	s.Equal("ok", resp.Body.String())
}

// ============================================
// CREATE LOCKER TESTS
// ============================================

func (s *IntegrationTestSuite) TestCreateLocker_Success() {
	resp := s.makeRequest("POST", "/lockers", map[string]string{
		"name": "L-NEW",
	})
	s.Equal(http.StatusCreated, resp.Code)

	var result struct {
		ID     int64  `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	err := json.NewDecoder(resp.Body).Decode(&result)
	s.NoError(err)
	s.Greater(result.ID, int64(0))
	s.Equal("L-NEW", result.Name)
	s.Equal("available", result.Status)

	// Cleanup
	_, err = s.db.Exec("DELETE FROM locker WHERE name = 'L-NEW'")
	s.NoError(err)
}

func (s *IntegrationTestSuite) TestCreateLocker_DuplicateName() {
	// L-01 already exists in seed data
	resp := s.makeRequest("POST", "/lockers", map[string]string{
		"name": "L-01",
	})
	s.Equal(http.StatusConflict, resp.Code)

	var result struct {
		ErrorCode    string `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	}
	err := json.NewDecoder(resp.Body).Decode(&result)
	s.NoError(err)
	s.Equal("LOCKER_NAME_ALREADY_EXISTS", result.ErrorCode)
}

func (s *IntegrationTestSuite) TestCreateLocker_EmptyName() {
	resp := s.makeRequest("POST", "/lockers", map[string]string{
		"name": "",
	})
	s.Equal(http.StatusBadRequest, resp.Code)

	var result struct {
		ErrorCode    string `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	}
	err := json.NewDecoder(resp.Body).Decode(&result)
	s.NoError(err)
	s.Equal("VALIDATION_ERROR", result.ErrorCode)
}

func (s *IntegrationTestSuite) TestCreateLocker_MissingBody() {
	resp := s.makeRequest("POST", "/lockers", nil)
	s.Equal(http.StatusInternalServerError, resp.Code)
}

// ============================================
// LIST LOCKERS TESTS
// ============================================

func (s *IntegrationTestSuite) TestListLockers() {
	resp := s.makeRequest("GET", "/lockers", nil)
	s.Equal(http.StatusOK, resp.Code)

	var result struct {
		Lockers []map[string]interface{} `json:"lockers"`
	}
	err := json.NewDecoder(resp.Body).Decode(&result)
	s.NoError(err)
	s.Len(result.Lockers, 5)

	// Verify statuses
	s.Equal("available", result.Lockers[0]["status"])
	s.Equal("available", result.Lockers[1]["status"])
	s.Equal("available", result.Lockers[2]["status"])
	s.Equal("in_use", result.Lockers[3]["status"])
	s.Equal("maintenance", result.Lockers[4]["status"])
}

// ============================================
// SELECT LOCKER TESTS
// ============================================

func (s *IntegrationTestSuite) TestSelectLocker_Success() {
	resp := s.makeRequest("POST", "/lockers/1/select", nil)
	s.Equal(http.StatusOK, resp.Code)

	var result struct {
		SessionID  int64  `json:"session_id"`
		QRCodeData string `json:"qr_code_data"`
		QRCodePNG  string `json:"qr_code_png"`
	}
	err := json.NewDecoder(resp.Body).Decode(&result)
	s.NoError(err)
	s.Greater(result.SessionID, int64(0))
	s.Equal("lntb1u1testinvoice", result.QRCodeData)
	s.NotEmpty(result.QRCodePNG)

	// Verify locker is now in_use
	lockersResp := s.makeRequest("GET", "/lockers", nil)
	var lockersResult struct {
		Lockers []map[string]interface{} `json:"lockers"`
	}
	err = json.NewDecoder(lockersResp.Body).Decode(&lockersResult)
	s.NoError(err)
	s.Equal("in_use", lockersResult.Lockers[0]["status"])

	// Cleanup: reset locker status
	_, err = s.db.Exec("UPDATE locker SET status = 'available' WHERE id = 1")
	s.NoError(err)
	// Cleanup: delete session
	_, err = s.db.Exec("DELETE FROM session WHERE locker_id = 1")
	s.NoError(err)
}

func (s *IntegrationTestSuite) TestSelectLocker_NotFound() {
	resp := s.makeRequest("POST", "/lockers/999/select", nil)
	s.Equal(http.StatusNotFound, resp.Code)

	var result struct {
		ErrorCode    string `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	}
	err := json.NewDecoder(resp.Body).Decode(&result)
	s.NoError(err)
	s.Equal("LOCKER_NOT_FOUND", result.ErrorCode)
}

func (s *IntegrationTestSuite) TestSelectLocker_NotAvailable() {
	// Locker 4 is in_use
	resp := s.makeRequest("POST", "/lockers/4/select", nil)
	s.Equal(http.StatusConflict, resp.Code)

	var result struct {
		ErrorCode    string `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	}
	err := json.NewDecoder(resp.Body).Decode(&result)
	s.NoError(err)
	s.Equal("LOCKER_NOT_AVAILABLE", result.ErrorCode)
}

func (s *IntegrationTestSuite) TestSelectLocker_Maintenance() {
	// Locker 5 is in maintenance
	resp := s.makeRequest("POST", "/lockers/5/select", nil)
	s.Equal(http.StatusConflict, resp.Code)

	var result struct {
		ErrorCode    string `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	}
	err := json.NewDecoder(resp.Body).Decode(&result)
	s.NoError(err)
	s.Equal("LOCKER_NOT_AVAILABLE", result.ErrorCode)
}

func (s *IntegrationTestSuite) TestSelectLocker_InvalidID() {
	resp := s.makeRequest("POST", "/lockers/abc/select", nil)
	s.Equal(http.StatusInternalServerError, resp.Code)
}

// ============================================
// CONFIRM PAYMENT TESTS
// ============================================

func (s *IntegrationTestSuite) TestConfirmPayment_Success() {
	// First, select a locker to create a session
	selectResp := s.makeRequest("POST", "/lockers/2/select", nil)
	s.Equal(http.StatusOK, selectResp.Code)

	var selectResult struct {
		SessionID int64 `json:"session_id"`
	}
	err := json.NewDecoder(selectResp.Body).Decode(&selectResult)
	s.NoError(err)

	// Confirm payment
	confirmResp := s.makeRequest("POST", fmt.Sprintf("/sessions/%d/confirm-payment", selectResult.SessionID), nil)
	s.Equal(http.StatusOK, confirmResp.Code)

	var confirmResult struct {
		SessionID int64  `json:"session_id"`
		Status    string `json:"status"`
		StartedAt string `json:"started_at"`
		ExpiredAt string `json:"expired_at"`
	}
	err = json.NewDecoder(confirmResp.Body).Decode(&confirmResult)
	s.NoError(err)
	s.Equal(selectResult.SessionID, confirmResult.SessionID)
	s.Equal("charging", confirmResult.Status)
	s.NotEmpty(confirmResult.StartedAt)
	s.NotEmpty(confirmResult.ExpiredAt)

	// Cleanup
	_, err = s.db.Exec("DELETE FROM session WHERE locker_id = 2")
	s.NoError(err)
	_, err = s.db.Exec("UPDATE locker SET status = 'available' WHERE id = 2")
	s.NoError(err)
}

func (s *IntegrationTestSuite) TestConfirmPayment_NotFound() {
	resp := s.makeRequest("POST", "/sessions/999/confirm-payment", nil)
	s.Equal(http.StatusNotFound, resp.Code)

	var result struct {
		ErrorCode    string `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	}
	err := json.NewDecoder(resp.Body).Decode(&result)
	s.NoError(err)
	s.Equal("SESSION_NOT_FOUND", result.ErrorCode)
}

func (s *IntegrationTestSuite) TestConfirmPayment_AlreadyPaid() {
	// Select locker and confirm payment
	selectResp := s.makeRequest("POST", "/lockers/3/select", nil)
	s.Equal(http.StatusOK, selectResp.Code)

	var selectResult struct {
		SessionID int64 `json:"session_id"`
	}
	err := json.NewDecoder(selectResp.Body).Decode(&selectResult)
	s.NoError(err)

	// First confirmation
	confirmResp := s.makeRequest("POST", fmt.Sprintf("/sessions/%d/confirm-payment", selectResult.SessionID), nil)
	s.Equal(http.StatusOK, confirmResp.Code)

	// Second confirmation should fail
	confirmResp2 := s.makeRequest("POST", fmt.Sprintf("/sessions/%d/confirm-payment", selectResult.SessionID), nil)
	s.Equal(http.StatusConflict, confirmResp2.Code)

	var result struct {
		ErrorCode    string `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	}
	err = json.NewDecoder(confirmResp2.Body).Decode(&result)
	s.NoError(err)
	s.Equal("SESSION_ALREADY_PAID", result.ErrorCode)

	// Cleanup
	_, err = s.db.Exec("DELETE FROM session WHERE locker_id = 3")
	s.NoError(err)
	_, err = s.db.Exec("UPDATE locker SET status = 'available' WHERE id = 3")
	s.NoError(err)
}

// ============================================
// CHECK SESSION TESTS
// ============================================

func (s *IntegrationTestSuite) TestCheckSession_Success() {
	// Select locker to create session
	selectResp := s.makeRequest("POST", "/lockers/1/select", nil)
	s.Equal(http.StatusOK, selectResp.Code)

	var selectResult struct {
		SessionID int64 `json:"session_id"`
	}
	err := json.NewDecoder(selectResp.Body).Decode(&selectResult)
	s.NoError(err)

	// Check session
	checkResp := s.makeRequest("GET", fmt.Sprintf("/sessions/%d", selectResult.SessionID), nil)
	s.Equal(http.StatusOK, checkResp.Code)

	var checkResult struct {
		ID         int64  `json:"id"`
		LockerID   int64  `json:"locker_id"`
		Status     string `json:"status"`
		QRCodeData string `json:"qr_code_data"`
		Amount     int64  `json:"amount"`
	}
	err = json.NewDecoder(checkResp.Body).Decode(&checkResult)
	s.NoError(err)
	s.Equal(selectResult.SessionID, checkResult.ID)
	s.Equal(int64(1), checkResult.LockerID)
	s.Equal("pending_payment", checkResult.Status)
	s.Equal(int64(2000), checkResult.Amount)

	// Cleanup
	_, err = s.db.Exec("DELETE FROM session WHERE locker_id = 1")
	s.NoError(err)
	_, err = s.db.Exec("UPDATE locker SET status = 'available' WHERE id = 1")
	s.NoError(err)
}

func (s *IntegrationTestSuite) TestCheckSession_NotFound() {
	resp := s.makeRequest("GET", "/sessions/999", nil)
	s.Equal(http.StatusNotFound, resp.Code)

	var result struct {
		ErrorCode    string `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	}
	err := json.NewDecoder(resp.Body).Decode(&result)
	s.NoError(err)
	s.Equal("SESSION_NOT_FOUND", result.ErrorCode)
}

func (s *IntegrationTestSuite) TestCheckSession_AutoExpire() {
	// Select locker and confirm payment
	selectResp := s.makeRequest("POST", "/lockers/1/select", nil)
	s.Equal(http.StatusOK, selectResp.Code)

	var selectResult struct {
		SessionID int64 `json:"session_id"`
	}
	err := json.NewDecoder(selectResp.Body).Decode(&selectResult)
	s.NoError(err)

	// Confirm payment
	confirmResp := s.makeRequest("POST", fmt.Sprintf("/sessions/%d/confirm-payment", selectResult.SessionID), nil)
	s.Equal(http.StatusOK, confirmResp.Code)

	// Manually set expired_at to the past to simulate expiry
	_, err = s.db.Exec("UPDATE session SET expired_at = datetime('now', '-1 hour') WHERE id = ?", selectResult.SessionID)
	s.NoError(err)

	// Check session - should auto-expire
	checkResp := s.makeRequest("GET", fmt.Sprintf("/sessions/%d", selectResult.SessionID), nil)
	s.Equal(http.StatusOK, checkResp.Code)

	var checkResult struct {
		Status string `json:"status"`
	}
	err = json.NewDecoder(checkResp.Body).Decode(&checkResult)
	s.NoError(err)
	s.Equal("completed", checkResult.Status)

	// Verify locker is back to available
	lockersResp := s.makeRequest("GET", "/lockers", nil)
	var lockersResult struct {
		Lockers []map[string]interface{} `json:"lockers"`
	}
	err = json.NewDecoder(lockersResp.Body).Decode(&lockersResult)
	s.NoError(err)
	s.Equal("available", lockersResult.Lockers[0]["status"])

	// Cleanup
	_, err = s.db.Exec("DELETE FROM session WHERE locker_id = 1")
	s.NoError(err)
}

// ============================================
// FULL E2E FLOW TEST
// ============================================

func (s *IntegrationTestSuite) TestFullFlow() {
	// 1. List lockers - verify initial state
	lockersResp := s.makeRequest("GET", "/lockers", nil)
	s.Equal(http.StatusOK, lockersResp.Code)

	// 2. Select locker
	selectResp := s.makeRequest("POST", "/lockers/2/select", nil)
	s.Equal(http.StatusOK, selectResp.Code)

	var selectResult struct {
		SessionID  int64  `json:"session_id"`
		QRCodeData string `json:"qr_code_data"`
		QRCodePNG  string `json:"qr_code_png"`
	}
	err := json.NewDecoder(selectResp.Body).Decode(&selectResult)
	s.NoError(err)
	s.Greater(selectResult.SessionID, int64(0))
	s.NotEmpty(selectResult.QRCodeData)
	s.NotEmpty(selectResult.QRCodePNG)

	// 3. Verify locker is in_use
	lockersResp2 := s.makeRequest("GET", "/lockers", nil)
	var lockersResult2 struct {
		Lockers []map[string]interface{} `json:"lockers"`
	}
	err = json.NewDecoder(lockersResp2.Body).Decode(&lockersResult2)
	s.NoError(err)
	s.Equal("in_use", lockersResult2.Lockers[1]["status"])

	// 4. Confirm payment
	confirmResp := s.makeRequest("POST", fmt.Sprintf("/sessions/%d/confirm-payment", selectResult.SessionID), nil)
	s.Equal(http.StatusOK, confirmResp.Code)

	var confirmResult struct {
		Status    string `json:"status"`
		StartedAt string `json:"started_at"`
		ExpiredAt string `json:"expired_at"`
	}
	err = json.NewDecoder(confirmResp.Body).Decode(&confirmResult)
	s.NoError(err)
	s.Equal("charging", confirmResult.Status)

	// 5. Check session - should be charging
	checkResp := s.makeRequest("GET", fmt.Sprintf("/sessions/%d", selectResult.SessionID), nil)
	s.Equal(http.StatusOK, checkResp.Code)

	var checkResult struct {
		Status string `json:"status"`
	}
	err = json.NewDecoder(checkResp.Body).Decode(&checkResult)
	s.NoError(err)
	s.Equal("charging", checkResult.Status)

	// Cleanup
	_, err = s.db.Exec("DELETE FROM session WHERE locker_id = 2")
	s.NoError(err)
	_, err = s.db.Exec("UPDATE locker SET status = 'available' WHERE id = 2")
	s.NoError(err)
}

// ============================================
// TEST SUITE RUNNER
// ============================================

func TestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
