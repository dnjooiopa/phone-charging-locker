package domain

import "time"

type SessionStatus string

const (
	SessionStatusPendingPayment SessionStatus = "pending_payment"
	SessionStatusCharging       SessionStatus = "charging"
	SessionStatusCompleted      SessionStatus = "completed"
	SessionStatusExpired        SessionStatus = "expired"
)

func (s SessionStatus) Valid() bool {
	switch s {
	case SessionStatusPendingPayment, SessionStatusCharging,
		SessionStatusCompleted, SessionStatusExpired:
		return true
	default:
		return false
	}
}

type Session struct {
	ID         int64         `json:"id"`
	LockerID   int64         `json:"locker_id"`
	Status     SessionStatus `json:"status"`
	QRCodeData string        `json:"qr_code_data"`
	Amount     int64         `json:"amount"`
	StartedAt  *time.Time    `json:"started_at"`
	ExpiredAt  *time.Time    `json:"expired_at"`
	CreatedAt  time.Time     `json:"-"`
	UpdatedAt  time.Time     `json:"-"`
}
