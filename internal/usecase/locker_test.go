package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dnjooiopa/phone-charging-locker/internal/domain"
)

func TestUsecase_CreateLocker(t *testing.T) {
	t.Run("validation", func(t *testing.T) {
		uc := &Usecase{}

		testCases := []struct {
			name             string
			params           *CreateLockerParams
			expectedErrorMsg string
		}{
			{
				name:             "name is empty",
				params:           &CreateLockerParams{Name: ""},
				expectedErrorMsg: "name is required",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := uc.CreateLocker(context.Background(), tc.params)
				if assert.Error(t, err) {
					assert.Equal(t, tc.expectedErrorMsg, err.Error())
				}
				assert.Nil(t, result)
			})
		}
	})

	t.Run("integration", func(t *testing.T) {
		testCases := []struct {
			name                 string
			params               *CreateLockerParams
			mockLockerRepository func(m *MockLockerRepository)
			expectedError        error
			expectResult         bool
		}{
			{
				name:   "success",
				params: &CreateLockerParams{Name: "L-01"},
				mockLockerRepository: func(m *MockLockerRepository) {
					m.On("Create", mock.Anything, "L-01").Return(int64(1), nil)
					m.On("FindByID", mock.Anything, int64(1)).Return(&domain.Locker{
						ID: 1, Name: "L-01", Status: domain.LockerStatusAvailable,
					}, nil)
				},
				expectResult: true,
			},
			{
				name:   "duplicate name",
				params: &CreateLockerParams{Name: "L-01"},
				mockLockerRepository: func(m *MockLockerRepository) {
					m.On("Create", mock.Anything, "L-01").Return(int64(0), ErrLockerNameAlreadyExists)
				},
				expectedError: ErrLockerNameAlreadyExists,
			},
		}

		for _, tc := range testCases {
			lockerRepo := &MockLockerRepository{}

			if tc.mockLockerRepository != nil {
				tc.mockLockerRepository(lockerRepo)
			}

			uc := &Usecase{
				lockerRepository: lockerRepo,
			}

			t.Run(tc.name, func(t *testing.T) {
				result, err := uc.CreateLocker(context.Background(), tc.params)
				if tc.expectedError != nil {
					assert.Error(t, err)
					assert.ErrorIs(t, err, tc.expectedError)
					assert.Nil(t, result)
				} else {
					assert.NoError(t, err)
					require.NotNil(t, result)
					assert.Equal(t, int64(1), result.Locker.ID)
					assert.Equal(t, "L-01", result.Locker.Name)
					assert.Equal(t, domain.LockerStatusAvailable, result.Locker.Status)
				}
			})
		}
	})
}

func TestUsecase_ListLockers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		lockerRepo := &MockLockerRepository{}
		lockerRepo.On("FindAll", mock.Anything).Return([]*domain.Locker{
			{ID: 1, Name: "L-01", Status: domain.LockerStatusAvailable},
			{ID: 2, Name: "L-02", Status: domain.LockerStatusInUse},
		}, nil)

		uc := &Usecase{
			lockerRepository: lockerRepo,
		}

		result, err := uc.ListLockers(context.Background())
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "L-01", result[0].Name)
		assert.Equal(t, domain.LockerStatusAvailable, result[0].Status)
		assert.Equal(t, "L-02", result[1].Name)
		assert.Equal(t, domain.LockerStatusInUse, result[1].Status)
	})

	t.Run("empty", func(t *testing.T) {
		lockerRepo := &MockLockerRepository{}
		lockerRepo.On("FindAll", mock.Anything).Return([]*domain.Locker{}, nil)

		uc := &Usecase{
			lockerRepository: lockerRepo,
		}

		result, err := uc.ListLockers(context.Background())
		assert.NoError(t, err)
		assert.Len(t, result, 0)
	})
}
