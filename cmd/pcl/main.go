package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/acoshift/configfile"
	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"

	"github.com/dnjooiopa/phone-charging-locker/internal/repository/invoice_repository"
	"github.com/dnjooiopa/phone-charging-locker/internal/repository/locker_repository"
	"github.com/dnjooiopa/phone-charging-locker/internal/repository/session_repository"
	"github.com/dnjooiopa/phone-charging-locker/internal/server/gin_server"
	"github.com/dnjooiopa/phone-charging-locker/internal/usecase"
	"github.com/dnjooiopa/phone-charging-locker/pkg/dbctx"
	"github.com/dnjooiopa/phone-charging-locker/schema"
)

type Config struct {
	Environment         string
	HOST                string
	PORT                int
	DBPath              string
	ChargingDuration    time.Duration
	ChargingAmount      int64
	PhoenixdProxyURL    string
	PhoenixdProxyAPIKey string
	WebhookURL          string
}

func newConfig() *Config {
	configfile.LoadDotEnv()
	cfg := configfile.NewEnvReader()

	dbPath := cfg.StringDefault("DB_PATH", "./tmp/data/pcl.db")

	if cfg.String("ENVIRONMENT") == "production" {
		dbPath = "/app/data/pcl.db"
	}

	return &Config{
		Environment:         cfg.String("ENVIRONMENT"),
		HOST:                cfg.String("HOST"),
		PORT:                cfg.Int("PORT"),
		DBPath:              dbPath,
		ChargingDuration:    cfg.DurationDefault("CHARGING_DURATION", 1*time.Hour),
		ChargingAmount:      cfg.Int64Default("CHARGING_AMOUNT", 2000),
		PhoenixdProxyURL:    cfg.String("PHOENIXD_PROXY_URL"),
		PhoenixdProxyAPIKey: cfg.String("PHOENIXD_PROXY_API_KEY"),
		WebhookURL:          cfg.String("WEBHOOK_URL"),
	}
}

func main() {
	cfg := newConfig()

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		log.Fatalln("cannot create data directory:", err)
	}

	db, err := sql.Open("sqlite", cfg.DBPath)
	if err != nil {
		log.Fatalln("cannot open SQLite database:", err)
	}

	// Enable WAL mode and foreign keys
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Fatalln("cannot set WAL mode:", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		log.Fatalln("cannot enable foreign keys:", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalln("cannot connect to database:", err)
	}
	log.Println("database connected")

	// Run migrations
	if err := schema.Migrate(context.Background(), db); err != nil {
		log.Fatalln("migration failed:", err)
	}
	log.Println("database migrated")

	lockerRepository := locker_repository.New()
	sessionRepository := session_repository.New()
	invoiceRepository := invoice_repository.NewPhoenixd(cfg.PhoenixdProxyURL, cfg.PhoenixdProxyAPIKey)

	uc := usecase.New(
		&usecase.Config{
			ChargingDuration: cfg.ChargingDuration,
			ChargingAmount:   cfg.ChargingAmount,
		},
		lockerRepository,
		sessionRepository,
		invoiceRepository,
	)

	if err := invoiceRepository.RegisterWebhookEndpoint(context.Background(), cfg.WebhookURL); err != nil {
		log.Fatalf("failed to register webhook endpoint: %v", err)
	}
	log.Println("webhook endpoint registered")

	server := gin_server.New(&gin_server.Config{
		Environment: cfg.Environment,
	}, uc)
	server.Use(gin_server.LoggerMiddleware())
	server.Use(gin.Recovery())
	server.Use(gin_server.ErrorHandler())
	server.Use(gin_server.DatabaseMiddleware(db))

	server.SetUpRoutes()

	// Start background session expiry worker
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx := dbctx.NewContext(context.Background(), db)
				count, err := uc.ExpireSessions(ctx)
				if err != nil {
					log.Printf("expire sessions error: %v", err)
				}
				if count > 0 {
					log.Printf("expired %d sessions", count)
				}
			case <-server.OngoingCtx().Done():
				return
			}
		}
	}()

	addr := net.JoinHostPort(cfg.HOST, strconv.Itoa(cfg.PORT))
	if err := server.Start(addr); err != nil {
		log.Fatalf("server start failed: %v", err)
	}
}
