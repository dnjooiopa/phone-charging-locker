package usecase

import "errors"

var (
	ErrLockerNotFound      = errors.New("locker not found")
	ErrLockerNotAvailable  = errors.New("locker not available")
	ErrSessionNotFound     = errors.New("session not found")
	ErrSessionAlreadyPaid  = errors.New("session already paid")
	ErrSessionExpired      = errors.New("session expired")
	ErrInvalidSessionState = errors.New("invalid session state")
)
