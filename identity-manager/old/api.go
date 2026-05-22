package api

import (
	"net/http"

	"portal-core/internal/constants"
	"portal-core/internal/handler"
	"portal-core/internal/middleware"
	"portal-core/internal/service"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Health             *handler.HealthHandler
	Jira               *handler.JiraHandler
	Metrics            *handler.MetricsHandler
	Notification       *handler.NotificationHandler
	User               *handler.UserHandler
	Mnemonic           *handler.MnemonicHandler
	Reason             *handler.ReasonHandler
	ReasonPrerequisite *handler.ReasonPrerequisiteHandler
	Feedback           *handler.FeedbackHandler
	SLA                *handler.SLAHandler
	Auth               *handler.AuthHandler
	Product            *handler.ProductHandler
	GitLinks           *handler.GitLinksHandler
	AIAssessment       *handler.AIAssessmentHandler
	Journal            *handler.JournalHandler
}

func Routes(
	r *gin.Engine,
	h *Handlers,
	jira *service.JiraService,
	sla service.SLAScheduler,
) {

	r.GET("/oidc/auth", h.Auth.InitiateAuth)
	r.GET("/oidc/callback", h.Auth.HandleCallback)
	r.GET("/oidc/logout", h.Auth.HandleLogout)

	protected := r.Group("")
	protected.Use(middleware.AuthRequired(h.Auth.AuthService))

	v1 := protected.Group("/api/v1")

	// Auth
	v1.GET("/auth/user", h.Auth.GetCurrentUser)
	v1.POST("/auth/refresh", h.Auth.RefreshToken)

	// Health
	v1.GET("/health", h.Health.HealthCheck)

	// Search
	v1.GET("/mnemonics/search", h.Mnemonic.SearchMnemonics)
	v1.GET("/products/search", h.Product.SearchProducts)
	v1.GET("/gitlinks/search", h.GitLinks.SearchGitLinks)

	// Feedback
	v1.POST("/feedback", h.Feedback.CreateFeedback)

	// Reasons
	v1.GET("/reasons", h.Reason.GetReasons)
	v1.GET("/reasons/search", h.Reason.Search)
	v1.GET("/prerequisites", h.ReasonPrerequisite.Get)

	// Jira
	j := v1.Group("/jira")

	j.POST("/issues", h.Jira.CreateJiraIssue)
	j.POST("/newissue", h.Jira.CreateNewJiraIssue)
	j.GET("/tickets/:id", h.Jira.GetTicketByID)
	j.GET("/health", jiraHealth(jira))

	// AI
	ai := v1.Group("/ai")
	ai.POST("/assessment", h.AIAssessment.Create)

	// Metrics
	m := v1.Group("/metrics")

	mCsp := m.Group("")
	mCsp.Use(middleware.RequireAnyGroup(
		h.Auth.AuthService,
		constants.GroupAdmin,
		constants.GroupSec,
	))

	mCsp.GET("/bugs", h.Metrics.GetMetrics)
	mCsp.GET("/statistics/check", h.Metrics.GetCheckTypeMetrics)
	mCsp.GET("/statistics/departments", h.Metrics.GetDepartmentStatistics)

	// HR
	mHR := m.Group("/hr")
	mHR.Use(middleware.RequireAnyGroup(
		h.Auth.AuthService,
		constants.GroupAdmin,
		constants.GroupSec,
	))

	mHR.GET("/headcount", h.Metrics.GetHeadcount)
	mHR.GET("/fte", h.Metrics.GetFTE)

	// SLA
	slaGroup := m.Group("/sla")

	mCsp.GET("/sla", h.SLA.GetSLAMetrics)
	mCsp.GET("/sla/analysis", h.SLA.GetAnalysisMetrics)

	slaGroup.POST("/calculate", h.SLA.TriggerCalculation)
	slaGroup.GET("/health", sla.HealthCheck())

	// Observability
	strategyObserve := m.Group("")

	strategyObserve.Use(middleware.RequireAnyGroup(
		h.Auth.AuthService,
		constants.GroupAdmin,
		constants.GroupSec,
		constants.GroupAppSecLead,
		constants.GroupDbaAudit,
	))

	strategyObserve.GET("/observability", h.Metrics.GetObservabilityMetrics)
	strategyObserve.GET("/coverage", h.Metrics.GetCoverageMetrics)

	// Coverage
	c := m.Group("/coverage")

	c.Use(middleware.RequireAnyGroup(
		h.Auth.AuthService,
		constants.GroupAdmin,
		constants.GroupSec,
		constants.GroupAppSecLead,
		constants.GroupDbaAudit,
	))

	c.GET("/devsec", h.Metrics.GetGeneralCoverageDevsecMetrics)
	c.GET("/offsec", h.Metrics.GetCoatingOffsecMetrics)

	// SAST
	sast := c.Group("/sast")

	sast.Use(middleware.RequireAnyGroup(
		h.Auth.AuthService,
		constants.GroupAdmin,
		constants.GroupSec,
	))

	sast.GET("", h.Metrics.GetCoverageSastMetrics)
	sast.GET("/freq", h.Metrics.GetCoverageSastFreqMetrics)
	sast.GET("/mont", h.Metrics.GetCoverageSastMonMetrics)
	sast.GET("/valid", h.Metrics.GetCoverageSastValidMetrics)

	// Pentest
	pentest := c.Group("/pentest")
	pentest.Use(middleware.RequireAnyGroup(
		h.Auth.AuthService,
		constants.GroupAdmin,
		constants.GroupSec,
	))

	pentest.GET("", h.Metrics.GetCoveragePentestMetrics)
	pentest.GET("/moni", h.Metrics.GetCoveragePentestMoniMetrics)
	pentest.GET("/external", h.Metrics.GetCoveragePentestExternalMetrics)

	// Notifications
	n := v1.Group("/notifications")

	n.GET("/user/:user_id", h.Notification.GetUserNotifications)
	n.POST("", h.Notification.CreateNotification)
	n.PUT("/read", h.Notification.MarkNotificationAsRead)
	n.PUT("/read-all", h.Notification.MarkAllNotificationsAsRead)

	// Tickets
	t := v1.Group("/tickets")

	t.GET("/user/:user_id", h.User.GetUserTickets)
	t.GET("/user/:user_id/:id", h.User.GetUserTicket)

	// Journals
	jc := v1.Group("/journals")
	jc.GET("/:id", h.Journal.GetJournalById)

	// Admin
	admin := v1.Group("/admin")

	admin.Use(middleware.AuthRequired(h.Auth.AuthService))
	admin.Use(middleware.RequireAllGroups(
		h.Auth.AuthService,
		constants.GroupAdmin,
	))

	admin.GET("/jwks/status", h.Auth.GetJWKSStatus)
}

func jiraHealth(jira *service.JiraService) gin.HandlerFunc {
	return func(c *gin.Context) {

		if err := jira.TestConnection(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "error",
				"message": "Jira connection failed",
				"error":   err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Jira connection successful",
		})
	}
}
