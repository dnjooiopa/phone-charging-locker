package gin_server

import (
	"context"
	"log"
	"net"
	"net/http"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/dnjooiopa/phone-charging-locker/internal/usecase"
)

const (
	shutdownPeriod      = 15 * time.Second
	shutdownHardPeriod  = 3 * time.Second
	readinessDrainDelay = 5 * time.Second
)

type Config struct {
	Environment string
}

type Server struct {
	router          *gin.Engine
	httpServer      *http.Server
	isShuttingDown  atomic.Bool
	ongoingCtx      context.Context
	stopOngoingFunc context.CancelFunc

	usecase *usecase.Usecase
}

// New creates a new Gin server instance
func New(cfg *Config, uc *usecase.Usecase) *Server {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	ongoingCtx, stopOngoingFunc := context.WithCancel(context.Background())

	s := &Server{
		router:          router,
		ongoingCtx:      ongoingCtx,
		stopOngoingFunc: stopOngoingFunc,

		usecase: uc,
	}

	s.router.GET("/healthz", s.healthCheck)

	return s
}

// healthCheck handles the readiness endpoint
func (s *Server) healthCheck(c *gin.Context) {
	if s.isShuttingDown.Load() {
		c.String(http.StatusServiceUnavailable, "server is shutting down")
		return
	}
	c.String(http.StatusOK, "ok")
}

// Start starts the server with graceful shutdown support
func (s *Server) Start(addr string) error {
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Ensure in-flight requests aren't cancelled immediately on SIGTERM
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
		BaseContext: func(_ net.Listener) context.Context {
			return s.ongoingCtx
		},
	}

	// Start server in goroutine
	go func() {
		log.Printf("server starting on %s", addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed to start: %v", err)
		}
	}()

	<-rootCtx.Done()
	stop()
	s.isShuttingDown.Store(true)
	log.Println("received shutdown signal, shutting down.")

	// Give time for readiness check to propagate
	time.Sleep(readinessDrainDelay)
	log.Println("readiness check propagated, now waiting for ongoing requests to finish.")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownPeriod)
	defer cancel()
	err := s.httpServer.Shutdown(shutdownCtx)
	s.stopOngoingFunc()

	if err != nil {
		log.Println("failed to wait for ongoing requests to finish, waiting for forced cancellation.")
		time.Sleep(shutdownHardPeriod)
	}

	log.Println("server shut down gracefully.")
	return nil
}

// Use registers global middleware to the router
func (s *Server) Use(middleware ...gin.HandlerFunc) {
	s.router.Use(middleware...)
}

// Handler returns the underlying HTTP handler for testing purposes
func (s *Server) Handler() http.Handler {
	return s.router
}

// OngoingCtx returns the server's ongoing context for background workers
func (s *Server) OngoingCtx() context.Context {
	return s.ongoingCtx
}
