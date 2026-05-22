package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"portal-core/internal/api"
	"portal-core/internal/config"
	"portal-core/internal/database"
	"portal-core/internal/handler"
	"portal-core/internal/logger"
	"portal-core/internal/service"
)

type portalCore struct {
	cfg      *config.Config
	log      *logrus.Entry
	db       *database.Db
	handlers *api.Handlers

	jira *service.JiraService
	sla  service.SLAScheduler

	c *components
}

type components struct {
	notifService        *service.NotificationService
	ticketService       *service.TicketService
	aiAssessmentService *service.AIAssessmentService
	metricsService      *service.MetricsService
	userService         *service.UserService
	reasonService       *service.ReasonService
	mnemonicService     *service.MnemonicService
	feedbackService     *service.FeedbackService
	productService      *service.ProductService
	gitLinksService     *service.GitLinksService

	slaCalc service.JiraSLACalculator

	authService  *service.AuthService
	jwtValidator *service.JWTValidator
	jwksManager  *service.JWKSManager
}

func (p *portalCore) createServices(ctx context.Context, c *components) {
	p.jira = service.NewJiraService(
		p.cfg.Jira.BaseUrl,
		p.cfg.Jira.Token,
		p.cfg.Jira.ProjectKey,
		p.cfg.Jira.IssueType,
		p.cfg.Jira.PriorityName,
		p.cfg.Jira.PriorityId,
		p.cfg.Jira.EpicPriorityName,
		p.cfg.Jira.EpicPriorityId,
		p.cfg.Jira.StoryPriorityName,
		p.cfg.Jira.StoryPriorityId,
		p.cfg.Jira.SubTaskPriorityName,
		p.cfg.Jira.SubTaskPriorityId,
		p.cfg.Jira.EpicParentLinkKey,
		p.cfg.Jira.EpicLinkField,
	)

	p.checkJiraConnection()

	notifAPI := service.NewNotificationAPIService(
		p.cfg.Notification.TriggerUrl,
		p.cfg.Notification.BasicAuthUser,
		p.cfg.Notification.BasicAuthPass,
		p.cfg.Notification.MaxRetries,
		int(p.cfg.Notification.Timeout.Seconds()),
	)

	c.notifService = service.NewNotificationService(
		p.db.Notify,
		notifAPI,
		p.db.Users,
		p.db.Jira,
	)

	ticketNotif := service.NewTicketNotificationService(
		p.db.Jira,
		c.notifService,
	)

	c.ticketService = service.NewTicketService(
		p.db.Jira,
		p.db.Reasons,
		p.db.Opers,
		p.jira,
		ticketNotif,
	)

	c.ticketService.SetDuplicateTimeout(
		p.cfg.App.DuplicateTimeout,
	)

	c.aiAssessmentService = service.NewAIAssessmentService(
		p.cfg.AIAssessment,
		p.jira,
		p.db.Jira,
		c.ticketService,
	)

	c.metricsService = service.NewMetricsService(
		p.db.Metrics,
	)

	c.userService = service.NewUserService(
		p.db.Users,
		p.db.Jira,
	)

	c.mnemonicService = service.NewMnemonicService(
		p.db.Mnemonic,
	)

	c.reasonService = service.NewReasonService(
		p.db.Reasons,
	)

	c.productService = service.NewProductService(
		p.db.Products,
	)

	c.gitLinksService = service.NewGitLinksService(
		p.db.GitLinks,
	)

	c.feedbackService = service.NewFeedbackService(
		p.db.Feedback,
		p.cfg.Feedback.TriggerUrl,
		p.cfg.Feedback.BasicAuthUser,
		p.cfg.Feedback.BasicAuthPass,
		p.cfg.Feedback.MaxRetries,
		int(p.cfg.Feedback.Timeout.Seconds()),
	)

	c.slaCalc = service.NewJiraSLACalculator(
		p.db.JiraSLA,
		p.db.SLA,
		p.cfg.App.SaveDetailedData,
	)

	p.sla = service.NewSLAScheduler(
		c.slaCalc,
	)

	if err := p.sla.Start(
		ctx,
		p.cfg.App.SlaUpdateHour,
	); err != nil {
		p.log.WithError(err).Warn("SLA scheduler")
	} else {
		p.log.WithField(
			"hour",
			p.cfg.App.SlaUpdateHour,
		).Debug("SLA scheduler started")
	}

	jwksURL := os.Getenv("IDP_JWKS_URL")
	if jwksURL == "" {
		p.log.Fatal("IDP_JWKS_URL environment variable is not set")
	}

	c.jwksManager = service.NewJWKSManager(jwksURL)
	c.jwksManager.SetCacheTTL(12 * time.Hour)

	c.jwtValidator = service.NewJWTValidator(
		c.jwksManager,
	)

	c.authService = service.NewAuthService(
		p.db.Sessions,
		c.userService,
		c.jwtValidator,
		c.jwksManager,
	)

	p.log.Debug("Services created")
}

func (p *portalCore) createHandlers(_ context.Context) {
	p.handlers = &api.Handlers{
		Health: handler.NewHealthHandler(
			p.db,
		),

		Jiras: handler.NewJiraHandler(
			p.jira,
			p.c.ticketService,
		),

		AIAssessments: handler.NewAIAssessmentHandler(
			p.c.aiAssessmentService,
			p.c.ticketService,
		),

		Metricss: handler.NewMetricsHandler(
			p.c.metricsService,
		),

		Notifications: handler.NewNotificationHandler(
			p.c.notifService,
		),

		Users: handler.NewUserHandler(
			p.c.userService,
		),

		Mnemonics: handler.NewMnemonicHandler(
			p.c.mnemonicService,
		),

		Feedbacks: handler.NewFeedbackHandler(
			p.c.feedbackService,
		),

		SLAS: handler.NewSLAHandler(
			p.c.slaCalc,
			p.sla,
		),

		Auths: handler.NewAuthHandler(
			p.c.authService,
		),

		Reasons: handler.NewReasonHandler(
			p.c.reasonService,
		),

		Products: handler.NewProductHandler(
			p.c.productService,
		),

		GitLinks: handler.NewGitLinksHandler(
			p.c.gitLinksService,
		),

		Journal: handler.NewJournalHandler(
			p.db.Journal,
		),
	}

	p.log.Debug("Handlers created")
}

func (p *portalCore) checkJiraConnection() {
	if err := p.jira.TestConnection(); err != nil {
		p.log.WithError(err).Warn("Jira connection")
	} else {
		p.log.Info("Jira connected")
	}
}

func (p *portalCore) run() {
	gin.SetMode(p.cfg.App.GinMode)

	r := gin.New()

	r.Use(gin.Recovery())

	if p.cfg.App.GinMode != gin.ReleaseMode {
		r.Use(gin.Logger())
	}

	api.Routes(
		r,
		p.handlers,
		p.jira,
		p.sla,
	)

	p.log.WithFields(logrus.Fields{
		"port":     "8088",
		"db":       p.cfg.Inventory.DbName,
		"jira_key": p.cfg.Jira.ProjectKey,
		"sla_hour": p.cfg.App.SlaUpdateHour,
		"gin_mode": p.cfg.App.GinMode,
	}).Info("Starting portal-core")

	srv := &http.Server{
		Addr:    ":8088",
		Handler: r,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if p.handlers.Auths != nil {
		go p.handlers.Auths.AuthService.StartLockCleanup(ctx)
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			p.log.Fatal(err)
		}
	}()

	p.shutdown(ctx, srv)
}

func (p *portalCore) shutdown(
	ctx context.Context,
	srv *http.Server,
) {
	ctx, stop := signal.NotifyContext(
		ctx,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	defer stop()

	<-ctx.Done()

	p.log.Info("Application portal-core shutting down...")

	if p.sla != nil {
		p.sla.Stop()
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)

	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		p.log.WithError(err).Error("Shutdown")
	}

	if p.db != nil {
		p.db.Close()
		p.log.Debug("DB connection closed")
	}

	p.log.Info("Application portal-core is disabled")
}

func main() {
	cfg := config.MustLoad()

	log := logger.New(cfg.App.Env)

	db, err := database.New(cfg)
	if err != nil {
		log.WithError(err).Fatal("database init")
	}

	app := &portalCore{
		cfg: cfg,
		log: log,
		db:  db,
		c:   &components{},
	}

	ctx := context.Background()

	app.createServices(ctx, app.c)
	app.createHandlers(ctx)
	app.run()
}
