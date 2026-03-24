package gin_server

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/moonrhythm/validator"

	"github.com/dnjooiopa/phone-charging-locker/internal/usecase"
)

type ErrorResponse struct {
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err

		log.Printf("error: %v", err)

		var (
			statusCode int
			errorCode  string
			message    string
		)

		var validateError *validator.Error
		if errors.As(err, &validateError) {
			statusCode = http.StatusBadRequest
			errorCode = "VALIDATION_ERROR"
			message = validateError.Error()
		} else {
			switch err {
			case usecase.ErrLockerNameAlreadyExists:
				statusCode = http.StatusConflict
				errorCode = "LOCKER_NAME_ALREADY_EXISTS"
				message = "locker name already exists"
			case usecase.ErrLockerNotFound:
				statusCode = http.StatusNotFound
				errorCode = "LOCKER_NOT_FOUND"
				message = "locker not found"
			case usecase.ErrLockerNotAvailable:
				statusCode = http.StatusConflict
				errorCode = "LOCKER_NOT_AVAILABLE"
				message = "locker not available"
			case usecase.ErrSessionNotFound:
				statusCode = http.StatusNotFound
				errorCode = "SESSION_NOT_FOUND"
				message = "session not found"
			case usecase.ErrSessionAlreadyPaid:
				statusCode = http.StatusConflict
				errorCode = "SESSION_ALREADY_PAID"
				message = "session already paid"
			case usecase.ErrSessionExpired:
				statusCode = http.StatusGone
				errorCode = "SESSION_EXPIRED"
				message = "session expired"
			case usecase.ErrInvalidSessionState:
				statusCode = http.StatusConflict
				errorCode = "INVALID_SESSION_STATE"
				message = "invalid session state"
			case usecase.ErrInvoiceCreationFailed:
				statusCode = http.StatusBadGateway
				errorCode = "INVOICE_CREATION_FAILED"
				message = "failed to create payment invoice"
			case usecase.ErrUnsupportedWebhookType:
				statusCode = http.StatusBadRequest
				errorCode = "UNSUPPORTED_WEBHOOK_TYPE"
				message = "unsupported webhook type"
			default:
				statusCode = http.StatusInternalServerError
				errorCode = "INTERNAL_SERVER_ERROR"
				message = "something went wrong"
			}
		}

		c.AbortWithStatusJSON(statusCode, ErrorResponse{
			ErrorCode:    errorCode,
			ErrorMessage: message,
		})
	}
}
