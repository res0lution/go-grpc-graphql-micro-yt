package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"identity-manager/internal/api"
	"identity-manager/internal/config"
	"identity-manager/internal/db"
	"identity-manager/internal/handler"
	"identity-manager/internal/logger"
	"identity-manager/internal/repository"
	"identity-manager/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

type Application struct {
	Config *config.Config
	Log    *logrus.Entry
	DB     *pgxpool.Pool
	Server *http.Server
}

func New(ctx context.Context) (*Application, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	log := logger.New(cfg.App.LogLevel)
	gin.SetMode(cfg.App.GinMode)

	pool, err := db.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("init postgres: %w", err)
	}

	// _ = client.NewIDPClient(cfg.IDP)
	// _ = client.NewCoreClient(cfg.Core)

	userRepo := repository.NewPostgresUserRepository(pool)
	sessionRepo := repository.NewPostgresSessionRepository(pool)

	authService := service.NewAuthService(*cfg, userRepo, sessionRepo)
	userService := service.NewUserService(userRepo, sessionRepo)
	sessionService := service.NewSessionService(sessionRepo, userRepo, authService)

	handlers := &api.Handlers{
		Health:            handler.NewHealthHandler(pool),
		Auth:              handler.NewAuthHandler(authService, cfg.Cookie),
		AuthSvc:           authService,
		CoreInternalToken: cfg.Core.InternalAuthToken,
		User:              handler.NewUserHandler(userService),
		Session:           handler.NewSessionHandler(sessionService),
		Internal:          handler.NewInternalIdentityHandler(sessionService),
	}

	router := api.NewRouter(log, handlers)
	srv := &http.Server{
		Addr:              ":" + cfg.App.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &Application{
		Config: cfg,
		Log:    log,
		DB:     pool,
		Server: srv,
	}, nil
}

func (a *Application) Close() {
	if a.DB != nil {
		a.DB.Close()
	}
}
