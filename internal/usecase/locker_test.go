package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/dnjooiopa/phone-charging-locker/internal/domain"
)

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
