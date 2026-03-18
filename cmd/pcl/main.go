package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"strconv"
	"time"

	"contrib.go.opencensus.io/integrations/ocsql"
	"github.com/acoshift/configfile"
	"github.com/acoshift/pgsql/pgctx"
	"github.com/gin-gonic/gin"

	"github.com/dnjooiopa/phone-charging-locker/internal/repository/locker_repository"
	"github.com/dnjooiopa/phone-charging-locker/internal/repository/session_repository"
	"github.com/dnjooiopa/phone-charging-locker/internal/server/gin_server"
	"github.com/dnjooiopa/phone-charging-locker/internal/usecase"
)

type Config struct {
	Environment      string
	HOST             string
	PORT             int
	DBURL            string
	ChargingDuration time.Duration
	ChargingAmount   int64
}

func newConfig() *Config {
	configfile.LoadDotEnv()
	cfg := configfile.NewEnvReader()
	return &Config{
		Environment:      cfg.String("ENVIRONMENT"),
		HOST:             cfg.String("HOST"),
		PORT:             cfg.Int("PORT"),
		DBURL:            cfg.String("DB_URL"),
		ChargingDuration: cfg.DurationDefault("CHARGING_DURATION", 1*time.Hour),
		ChargingAmount:   cfg.Int64Default("CHARGING_AMOUNT", 2000),
	}
}

func main() {
	cfg := newConfig()

	driver, _ := ocsql.Register("postgres")
	db, err := sql.Open(driver, cfg.DBURL)
	if err != nil {
		log.Fatalln("cannot open Postgres driver", err.Error())
	}

	err = db.Ping()
	if err != nil {
		log.Fatalln("cannot connect to Postgres:", err.Error())
	}
	log.Println("database connected")

	lockerRepository := locker_repository.NewPostgresDB()
	sessionRepository := session_repository.NewPostgresDB()

	uc := usecase.New(
		&usecase.Config{
			ChargingDuration: cfg.ChargingDuration,
			ChargingAmount:   cfg.ChargingAmount,
		},
		lockerRepository,
		sessionRepository,
	)

	server := gin_server.New(&gin_server.Config{
		Environment: cfg.Environment,
	}, uc)
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
				ctx := pgctx.NewContext(context.Background(), db)
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
