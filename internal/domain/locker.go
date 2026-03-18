package domain

import "time"

type LockerStatus string

const (
	LockerStatusAvailable   LockerStatus = "available"
	LockerStatusInUse       LockerStatus = "in_use"
	LockerStatusMaintenance LockerStatus = "maintenance"
)

func (s LockerStatus) Valid() bool {
	switch s {
	case LockerStatusAvailable, LockerStatusInUse, LockerStatusMaintenance:
		return true
	default:
		return false
	}
}

type Locker struct {
	ID        int64        `json:"id"`
	Name      string       `json:"name"`
	Status    LockerStatus `json:"status"`
	CreatedAt time.Time    `json:"-"`
	UpdatedAt time.Time    `json:"-"`
}
