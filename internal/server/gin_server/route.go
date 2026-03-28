package gin_server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dnjooiopa/phone-charging-locker/internal/domain"
	"github.com/dnjooiopa/phone-charging-locker/internal/usecase"
)

func (s *Server) SetUpRoutes() {
	r := s.router.Group("")
	r.POST("/lockers", s.CreateLocker)
	r.GET("/lockers", s.ListLockers)
	r.DELETE("/lockers/:id", s.DeleteLocker)
	r.POST("/lockers/:id/select", s.SelectLocker)
	r.GET("/sessions/:id", s.CheckSession)
	r.POST("/sessions/:id/confirm-payment", s.ConfirmPayment)
	r.POST("/webhook/phoenixd", s.HandleWebhook)
}

type CreateLockerRequest struct {
	Name string `json:"name"`
}

type CreateLockerResponse struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

func (s *Server) CreateLocker(c *gin.Context) {
	var req CreateLockerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}

	result, err := s.usecase.CreateLocker(c.Request.Context(), &usecase.CreateLockerParams{
		Name: req.Name,
	})
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, CreateLockerResponse{
		ID:     result.Locker.ID,
		Name:   result.Locker.Name,
		Status: string(result.Locker.Status),
	})
}

func (s *Server) DeleteLocker(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(err)
		return
	}

	err = s.usecase.DeleteLocker(c.Request.Context(), &usecase.DeleteLockerParams{
		LockerID: id,
	})
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

type LockerItem struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type ListLockersResponse struct {
	Lockers []*LockerItem `json:"lockers"`
}

func (s *Server) ListLockers(c *gin.Context) {
	lockers, err := s.usecase.ListLockers(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	items := make([]*LockerItem, len(lockers))
	for i, l := range lockers {
		items[i] = &LockerItem{
			ID:     l.ID,
			Name:   l.Name,
			Status: string(l.Status),
		}
	}

	c.JSON(http.StatusOK, ListLockersResponse{
		Lockers: items,
	})
}

type SelectLockerResponse struct {
	SessionID  int64  `json:"session_id"`
	QRCodeData string `json:"qr_code_data"`
	QRCodePNG  string `json:"qr_code_png"`
}

func (s *Server) SelectLocker(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(err)
		return
	}

	result, err := s.usecase.SelectLocker(c.Request.Context(), &usecase.SelectLockerParams{
		LockerID: id,
	})
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, SelectLockerResponse{
		SessionID:  result.SessionID,
		QRCodeData: result.QRCodeData,
		QRCodePNG:  result.QRCodePNG,
	})
}

type CheckSessionResponse struct {
	ID         int64                `json:"id"`
	LockerID   int64                `json:"locker_id"`
	Status     domain.SessionStatus `json:"status"`
	QRCodeData string               `json:"qr_code_data"`
	Amount     int64                `json:"amount"`
	StartedAt  *string              `json:"started_at"`
	ExpiredAt  *string              `json:"expired_at"`
}

func (s *Server) CheckSession(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(err)
		return
	}

	session, err := s.usecase.CheckSession(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}

	resp := CheckSessionResponse{
		ID:         session.ID,
		LockerID:   session.LockerID,
		Status:     session.Status,
		QRCodeData: session.QRCodeData,
		Amount:     session.Amount,
	}

	if session.StartedAt != nil {
		t := session.StartedAt.Format(time.RFC3339)
		resp.StartedAt = &t
	}
	if session.ExpiredAt != nil {
		t := session.ExpiredAt.Format(time.RFC3339)
		resp.ExpiredAt = &t
	}

	c.JSON(http.StatusOK, resp)
}

type ConfirmPaymentResponse struct {
	SessionID int64                `json:"session_id"`
	Status    domain.SessionStatus `json:"status"`
	StartedAt string               `json:"started_at"`
	ExpiredAt string               `json:"expired_at"`
}

func (s *Server) ConfirmPayment(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(err)
		return
	}

	session, err := s.usecase.ConfirmPayment(c.Request.Context(), &usecase.ConfirmPaymentParams{
		SessionID: id,
	})
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, ConfirmPaymentResponse{
		SessionID: session.ID,
		Status:    session.Status,
		StartedAt: session.StartedAt.Format(time.RFC3339),
		ExpiredAt: session.ExpiredAt.Format(time.RFC3339),
	})
}

type WebhookResponse struct {
	SessionID int64                `json:"session_id"`
	Status    domain.SessionStatus `json:"status"`
	StartedAt string               `json:"started_at"`
	ExpiredAt string               `json:"expired_at"`
}

func (s *Server) HandleWebhook(c *gin.Context) {
	var payload usecase.WebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.Error(err)
		return
	}

	session, err := s.usecase.HandleWebhook(c.Request.Context(), &payload)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, WebhookResponse{
		SessionID: session.ID,
		Status:    session.Status,
		StartedAt: session.StartedAt.Format(time.RFC3339),
		ExpiredAt: session.ExpiredAt.Format(time.RFC3339),
	})
}
