package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dnjooiopa/phone-charging-locker/internal/domain"
)

// MockLockerRepository is a mock implementation of LockerRepository
type MockLockerRepository struct {
	mock.Mock
}

func (m *MockLockerRepository) FindAll(ctx context.Context) ([]*domain.Locker, error) {
	args := m.Called(ctx)
	var r0 []*domain.Locker
	if args.Get(0) != nil {
		r0 = args.Get(0).([]*domain.Locker)
	}
	return r0, args.Error(1)
}

func (m *MockLockerRepository) FindByID(ctx context.Context, id int64) (*domain.Locker, error) {
	args := m.Called(ctx, id)
	var r0 *domain.Locker
	if args.Get(0) != nil {
		r0 = args.Get(0).(*domain.Locker)
	}
	return r0, args.Error(1)
}

func (m *MockLockerRepository) UpdateStatus(ctx context.Context, id int64, status domain.LockerStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

// MockSessionRepository is a mock implementation of SessionRepository
type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Create(ctx context.Context, session *domain.Session) (int64, error) {
	args := m.Called(ctx, session)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockSessionRepository) FindByID(ctx context.Context, id int64) (*domain.Session, error) {
	args := m.Called(ctx, id)
	var r0 *domain.Session
	if args.Get(0) != nil {
		r0 = args.Get(0).(*domain.Session)
	}
	return r0, args.Error(1)
}

func (m *MockSessionRepository) UpdateStatus(ctx context.Context, id int64, status domain.SessionStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockSessionRepository) UpdateInvoiceData(ctx context.Context, id int64, qrCodeData, paymentHash string) error {
	args := m.Called(ctx, id, qrCodeData, paymentHash)
	return args.Error(0)
}

func (m *MockSessionRepository) UpdatePaymentConfirmed(ctx context.Context, id int64, startedAt, expiredAt time.Time) error {
	args := m.Called(ctx, id, startedAt, expiredAt)
	return args.Error(0)
}

func (m *MockSessionRepository) FindByPaymentHash(ctx context.Context, paymentHash string) (*domain.Session, error) {
	args := m.Called(ctx, paymentHash)
	var r0 *domain.Session
	if args.Get(0) != nil {
		r0 = args.Get(0).(*domain.Session)
	}
	return r0, args.Error(1)
}

func (m *MockSessionRepository) FindExpiredChargingSessions(ctx context.Context, now time.Time) ([]*domain.Session, error) {
	args := m.Called(ctx, now)
	var r0 []*domain.Session
	if args.Get(0) != nil {
		r0 = args.Get(0).([]*domain.Session)
	}
	return r0, args.Error(1)
}

// MockInvoiceRepository is a mock implementation of InvoiceRepository
type MockInvoiceRepository struct {
	mock.Mock
}

func (m *MockInvoiceRepository) CreateInvoice(ctx context.Context, params *CreateInvoiceParams) (*CreateInvoiceResult, error) {
	args := m.Called(ctx, params)
	var r0 *CreateInvoiceResult
	if args.Get(0) != nil {
		r0 = args.Get(0).(*CreateInvoiceResult)
	}
	return r0, args.Error(1)
}

func TestUsecase_SelectLocker(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		uc := &Usecase{}

		testCases := []struct {
			name             string
			params           *SelectLockerParams
			expectedErrorMsg string
		}{
			{
				name:             "locker_id is zero",
				params:           &SelectLockerParams{LockerID: 0},
				expectedErrorMsg: "locker_id is required",
			},
			{
				name:             "locker_id is negative",
				params:           &SelectLockerParams{LockerID: -1},
				expectedErrorMsg: "locker_id is required",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := uc.SelectLocker(context.Background(), tc.params)
				if assert.Error(t, err) {
					assert.Equal(t, tc.expectedErrorMsg, err.Error())
				}
				assert.Nil(t, result)
			})
		}
	})

	t.Run("integration", func(t *testing.T) {
		testCases := []struct {
			name                  string
			params                *SelectLockerParams
			mockLockerRepository  func(m *MockLockerRepository)
			mockSessionRepository func(m *MockSessionRepository)
			mockInvoiceRepository func(m *MockInvoiceRepository)
			expectedError         error
			expectResult          bool
		}{
			{
				name:   "success",
				params: &SelectLockerParams{LockerID: 1},
				mockLockerRepository: func(m *MockLockerRepository) {
					m.On("FindByID", mock.Anything, int64(1)).Return(&domain.Locker{
						ID: 1, Name: "L-01", Status: domain.LockerStatusAvailable,
					}, nil)
					m.On("UpdateStatus", mock.Anything, int64(1), domain.LockerStatusInUse).Return(nil)
				},
				mockSessionRepository: func(m *MockSessionRepository) {
					m.On("Create", mock.Anything, mock.AnythingOfType("*domain.Session")).Return(int64(1), nil)
					m.On("UpdateInvoiceData", mock.Anything, int64(1), "lntb1u1fake", "abc123hash").Return(nil)
				},
				mockInvoiceRepository: func(m *MockInvoiceRepository) {
					m.On("CreateInvoice", mock.Anything, mock.AnythingOfType("*usecase.CreateInvoiceParams")).Return(&CreateInvoiceResult{
						PaymentHash: "abc123hash",
						Serialized:  "lntb1u1fake",
					}, nil)
				},
				expectedError: nil,
				expectResult:  true,
			},
			{
				name:   "invoice creation failed",
				params: &SelectLockerParams{LockerID: 1},
				mockLockerRepository: func(m *MockLockerRepository) {
					m.On("FindByID", mock.Anything, int64(1)).Return(&domain.Locker{
						ID: 1, Name: "L-01", Status: domain.LockerStatusAvailable,
					}, nil)
				},
				mockSessionRepository: func(m *MockSessionRepository) {
					m.On("Create", mock.Anything, mock.AnythingOfType("*domain.Session")).Return(int64(1), nil)
				},
				mockInvoiceRepository: func(m *MockInvoiceRepository) {
					m.On("CreateInvoice", mock.Anything, mock.AnythingOfType("*usecase.CreateInvoiceParams")).Return(nil, errors.New("connection refused"))
				},
				expectedError: ErrInvoiceCreationFailed,
			},
			{
				name:   "locker not found",
				params: &SelectLockerParams{LockerID: 999},
				mockLockerRepository: func(m *MockLockerRepository) {
					m.On("FindByID", mock.Anything, int64(999)).Return(nil, ErrLockerNotFound)
				},
				expectedError: ErrLockerNotFound,
			},
			{
				name:   "locker in use",
				params: &SelectLockerParams{LockerID: 2},
				mockLockerRepository: func(m *MockLockerRepository) {
					m.On("FindByID", mock.Anything, int64(2)).Return(&domain.Locker{
						ID: 2, Name: "L-02", Status: domain.LockerStatusInUse,
					}, nil)
				},
				expectedError: ErrLockerNotAvailable,
			},
			{
				name:   "locker in maintenance",
				params: &SelectLockerParams{LockerID: 3},
				mockLockerRepository: func(m *MockLockerRepository) {
					m.On("FindByID", mock.Anything, int64(3)).Return(&domain.Locker{
						ID: 3, Name: "L-03", Status: domain.LockerStatusMaintenance,
					}, nil)
				},
				expectedError: ErrLockerNotAvailable,
			},
		}

		for _, tc := range testCases {
			lockerRepo := &MockLockerRepository{}
			sessionRepo := &MockSessionRepository{}
			invoiceRepo := &MockInvoiceRepository{}

			if tc.mockLockerRepository != nil {
				tc.mockLockerRepository(lockerRepo)
			}
			if tc.mockSessionRepository != nil {
				tc.mockSessionRepository(sessionRepo)
			}
			if tc.mockInvoiceRepository != nil {
				tc.mockInvoiceRepository(invoiceRepo)
			}

			uc := &Usecase{
				config: &Config{
					ChargingDuration: 1 * time.Hour,
					ChargingAmount:   2000,
				},
				lockerRepository:  lockerRepo,
				sessionRepository: sessionRepo,
				invoiceRepository:    invoiceRepo,
			}

			t.Run(tc.name, func(t *testing.T) {
				result, err := uc.SelectLocker(context.Background(), tc.params)
				if tc.expectedError != nil {
					assert.Error(t, err)
					assert.ErrorIs(t, err, tc.expectedError)
					assert.Nil(t, result)
				} else {
					assert.NoError(t, err)
					require.NotNil(t, result)
					assert.Equal(t, int64(1), result.SessionID)
					assert.Equal(t, "lntb1u1fake", result.QRCodeData)
					assert.NotEmpty(t, result.QRCodePNG)
				}
			})
		}
	})
}

func TestUsecase_ConfirmPayment(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		uc := &Usecase{}

		testCases := []struct {
			name             string
			params           *ConfirmPaymentParams
			expectedErrorMsg string
		}{
			{
				name:             "session_id is zero",
				params:           &ConfirmPaymentParams{SessionID: 0},
				expectedErrorMsg: "session_id is required",
			},
			{
				name:             "session_id is negative",
				params:           &ConfirmPaymentParams{SessionID: -1},
				expectedErrorMsg: "session_id is required",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := uc.ConfirmPayment(context.Background(), tc.params)
				if assert.Error(t, err) {
					assert.Equal(t, tc.expectedErrorMsg, err.Error())
				}
				assert.Nil(t, result)
			})
		}
	})

	t.Run("integration", func(t *testing.T) {
		testCases := []struct {
			name                  string
			params                *ConfirmPaymentParams
			mockSessionRepository func(m *MockSessionRepository)
			expectedError         error
		}{
			{
				name:   "success",
				params: &ConfirmPaymentParams{SessionID: 1},
				mockSessionRepository: func(m *MockSessionRepository) {
					m.On("FindByID", mock.Anything, int64(1)).Return(&domain.Session{
						ID:       1,
						LockerID: 1,
						Status:   domain.SessionStatusPendingPayment,
					}, nil)
					m.On("UpdatePaymentConfirmed", mock.Anything, int64(1), mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).Return(nil)
				},
				expectedError: nil,
			},
			{
				name:   "session not found",
				params: &ConfirmPaymentParams{SessionID: 999},
				mockSessionRepository: func(m *MockSessionRepository) {
					m.On("FindByID", mock.Anything, int64(999)).Return(nil, ErrSessionNotFound)
				},
				expectedError: ErrSessionNotFound,
			},
			{
				name:   "session already paid",
				params: &ConfirmPaymentParams{SessionID: 2},
				mockSessionRepository: func(m *MockSessionRepository) {
					m.On("FindByID", mock.Anything, int64(2)).Return(&domain.Session{
						ID:     2,
						Status: domain.SessionStatusCharging,
					}, nil)
				},
				expectedError: ErrSessionAlreadyPaid,
			},
			{
				name:   "session completed",
				params: &ConfirmPaymentParams{SessionID: 3},
				mockSessionRepository: func(m *MockSessionRepository) {
					m.On("FindByID", mock.Anything, int64(3)).Return(&domain.Session{
						ID:     3,
						Status: domain.SessionStatusCompleted,
					}, nil)
				},
				expectedError: ErrInvalidSessionState,
			},
		}

		for _, tc := range testCases {
			sessionRepo := &MockSessionRepository{}

			if tc.mockSessionRepository != nil {
				tc.mockSessionRepository(sessionRepo)
			}

			uc := &Usecase{
				config: &Config{
					ChargingDuration: 1 * time.Hour,
					ChargingAmount:   2000,
				},
				sessionRepository: sessionRepo,
			}

			t.Run(tc.name, func(t *testing.T) {
				result, err := uc.ConfirmPayment(context.Background(), tc.params)
				if tc.expectedError != nil {
					assert.Error(t, err)
					assert.ErrorIs(t, err, tc.expectedError)
					assert.Nil(t, result)
				} else {
					assert.NoError(t, err)
					require.NotNil(t, result)
					assert.Equal(t, domain.SessionStatusCharging, result.Status)
					assert.NotNil(t, result.StartedAt)
					assert.NotNil(t, result.ExpiredAt)
				}
			})
		}
	})
}

func TestUsecase_CheckSession(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		uc := &Usecase{}

		testCases := []struct {
			name             string
			id               int64
			expectedErrorMsg string
		}{
			{
				name:             "id is zero",
				id:               0,
				expectedErrorMsg: "id is required",
			},
			{
				name:             "id is negative",
				id:               -1,
				expectedErrorMsg: "id is required",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := uc.CheckSession(context.Background(), tc.id)
				if assert.Error(t, err) {
					assert.Equal(t, tc.expectedErrorMsg, err.Error())
				}
				assert.Nil(t, result)
			})
		}
	})

	t.Run("integration", func(t *testing.T) {
		now := time.Now()
		pastTime := now.Add(-1 * time.Hour)
		futureTime := now.Add(1 * time.Hour)

		testCases := []struct {
			name                  string
			id                    int64
			mockLockerRepository  func(m *MockLockerRepository)
			mockSessionRepository func(m *MockSessionRepository)
			expectedError         error
			expectedStatus        domain.SessionStatus
		}{
			{
				name: "success - pending payment",
				id:   1,
				mockSessionRepository: func(m *MockSessionRepository) {
					m.On("FindByID", mock.Anything, int64(1)).Return(&domain.Session{
						ID:     1,
						Status: domain.SessionStatusPendingPayment,
					}, nil)
				},
				expectedStatus: domain.SessionStatusPendingPayment,
			},
			{
				name: "success - charging not expired",
				id:   2,
				mockSessionRepository: func(m *MockSessionRepository) {
					m.On("FindByID", mock.Anything, int64(2)).Return(&domain.Session{
						ID:        2,
						Status:    domain.SessionStatusCharging,
						ExpiredAt: &futureTime,
					}, nil)
				},
				expectedStatus: domain.SessionStatusCharging,
			},
			{
				name: "auto-expire on check",
				id:   3,
				mockLockerRepository: func(m *MockLockerRepository) {
					m.On("UpdateStatus", mock.Anything, int64(1), domain.LockerStatusAvailable).Return(nil)
				},
				mockSessionRepository: func(m *MockSessionRepository) {
					m.On("FindByID", mock.Anything, int64(3)).Return(&domain.Session{
						ID:        3,
						LockerID:  1,
						Status:    domain.SessionStatusCharging,
						ExpiredAt: &pastTime,
					}, nil)
					m.On("UpdateStatus", mock.Anything, int64(3), domain.SessionStatusCompleted).Return(nil)
				},
				expectedStatus: domain.SessionStatusCompleted,
			},
			{
				name: "session not found",
				id:   999,
				mockSessionRepository: func(m *MockSessionRepository) {
					m.On("FindByID", mock.Anything, int64(999)).Return(nil, ErrSessionNotFound)
				},
				expectedError: ErrSessionNotFound,
			},
		}

		for _, tc := range testCases {
			lockerRepo := &MockLockerRepository{}
			sessionRepo := &MockSessionRepository{}

			if tc.mockLockerRepository != nil {
				tc.mockLockerRepository(lockerRepo)
			}
			if tc.mockSessionRepository != nil {
				tc.mockSessionRepository(sessionRepo)
			}

			uc := &Usecase{
				lockerRepository:  lockerRepo,
				sessionRepository: sessionRepo,
			}

			t.Run(tc.name, func(t *testing.T) {
				result, err := uc.CheckSession(context.Background(), tc.id)
				if tc.expectedError != nil {
					assert.Error(t, err)
					assert.ErrorIs(t, err, tc.expectedError)
					assert.Nil(t, result)
				} else {
					assert.NoError(t, err)
					require.NotNil(t, result)
					assert.Equal(t, tc.expectedStatus, result.Status)
				}
			})
		}
	})
}

func TestUsecase_ExpireSessions(t *testing.T) {
	t.Run("expires multiple sessions", func(t *testing.T) {
		lockerRepo := &MockLockerRepository{}
		sessionRepo := &MockSessionRepository{}

		pastTime := time.Now().Add(-1 * time.Hour)
		sessions := []*domain.Session{
			{ID: 1, LockerID: 1, Status: domain.SessionStatusCharging, ExpiredAt: &pastTime},
			{ID: 2, LockerID: 2, Status: domain.SessionStatusCharging, ExpiredAt: &pastTime},
		}

		sessionRepo.On("FindExpiredChargingSessions", mock.Anything, mock.AnythingOfType("time.Time")).Return(sessions, nil)
		sessionRepo.On("UpdateStatus", mock.Anything, int64(1), domain.SessionStatusCompleted).Return(nil)
		sessionRepo.On("UpdateStatus", mock.Anything, int64(2), domain.SessionStatusCompleted).Return(nil)
		lockerRepo.On("UpdateStatus", mock.Anything, int64(1), domain.LockerStatusAvailable).Return(nil)
		lockerRepo.On("UpdateStatus", mock.Anything, int64(2), domain.LockerStatusAvailable).Return(nil)

		uc := &Usecase{
			lockerRepository:  lockerRepo,
			sessionRepository: sessionRepo,
		}

		count, err := uc.ExpireSessions(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("no sessions to expire", func(t *testing.T) {
		sessionRepo := &MockSessionRepository{}
		sessionRepo.On("FindExpiredChargingSessions", mock.Anything, mock.AnythingOfType("time.Time")).Return([]*domain.Session{}, nil)

		uc := &Usecase{
			sessionRepository: sessionRepo,
		}

		count, err := uc.ExpireSessions(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}
