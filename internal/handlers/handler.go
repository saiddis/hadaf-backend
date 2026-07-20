// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package handlers

import (
	"context"
	"errors"
	"net/http"

	"shb/internal/configs"
	"shb/internal/models"
	"shb/internal/repositories/filters"
	"shb/internal/services"
	"shb/pkg/constants"
	"shb/pkg/external/sms/smsProvider"
	"shb/pkg/middlewares"
	"shb/pkg/myerrors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/oauth2"
)

// Limiter defines the rate-limiting contract used by the handler layer.
type Limiter interface {
	Allow(ctx context.Context, key string, limit int, windowSeconds int) (bool, error)
	ResetAttempts(ctx context.Context, key string) error
}

// IService defines the business logic contract used by the handler layer.
type IService interface {
	SendOTP(ctx context.Context, receiver string) (int, error)
	ConfirmOTP(ctx context.Context, phone, otp string) (*models.TokenResponse, error)
	Login(ctx context.Context, phone, password string) (*models.TokenResponse, error)
	Register(ctx context.Context, email, phone, password, fullName, role string, institutionID *int) (*models.TokenResponse, error)
	GetUserByID(ctx context.Context, id int) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) error
	LoginOAuth(ctx context.Context, oauthUserInfo models.OAuthUserInfo) (*models.TokenResponse, error)
	UpdateUserOAuthInfoByEmail(ctx context.Context, info models.OAuthUserInfo) (*models.User, error)

	GetAllInstitutions(ctx context.Context, q models.InstitutionListQuery) (*models.InstitutionPage, error)
	CreateInstitution(ctx context.Context, i *models.Institution) (int, error)
	GetInstitutionByID(ctx context.Context, id int) (*models.Institution, error)

	CreateNeed(ctx context.Context, need *models.Need) (int, error)
	UpdateNeed(ctx context.Context, n *models.Need) error
	DeleteNeed(ctx context.Context, id int) error
	GetNeedByID(ctx context.Context, id int) (*models.Need, error)
	GetNeedsByInstitution(ctx context.Context, filter filters.NeedsFilter, institutionID int) ([]*models.Need, error)

	CreateBooking(ctx context.Context, userID, needID int, quantity float64, note string) (int, error)
	ApproveBooking(ctx context.Context, bookingID, institutionUserID int) error
	RejectBooking(ctx context.Context, bookingID, institutionUserID int) error
	CompleteBooking(ctx context.Context, bookingID, institutionUserID int) error
	GetBookingsByInstitution(ctx context.Context, institutionID int) ([]*models.Booking, error)
	GetBookingsByUser(ctx context.Context, userID int) ([]*models.Booking, error)
	CancelMyBooking(ctx context.Context, bookingID int, userID int) error
	UpdateMyBooking(ctx context.Context, bookingID int, userID int, qty float64) error
	UserExists(ctx context.Context, email string, phone string) (bool, bool, bool, error)

	// --- Event Methods ---
	CreateEvent(ctx context.Context, e *models.Event) (int, error)
	GetAllEvents(ctx context.Context, q models.EventListQuery) (*models.EventPage, error)
	GetEventDetail(ctx context.Context, q models.EventDetailQuery) (*models.EventResponse, error)
	GetEventByID(ctx context.Context, id int) (*models.Event, error)
	JoinEvent(ctx context.Context, eventID, userID int) error
	LeaveEvent(ctx context.Context, eventID, userID int) error
	GetInstitutionEvents(ctx context.Context, institutionID int) ([]*models.EventResponse, error)
	ApproveEvent(ctx context.Context, eventID int) error
	RejectEvent(ctx context.Context, eventID int) error

	// --- Stats ---
	GetPublicStats(ctx context.Context) (map[string]int, error)

	// --- Vacancies ---
	GetAllVacancies(ctx context.Context) ([]*models.Vacancy, error)
	GetVacancyByID(ctx context.Context, id int) (*models.Vacancy, error)

	// --- Team Members ---
	GetAllTeamMembers(ctx context.Context) ([]*models.TeamMember, error)
	GetTeamMemberByID(ctx context.Context, id int) (*models.TeamMember, error)

	// --- SMS ---
	CheckSMSBalance(ctx context.Context) (*smsProvider.BalanceResult, error)

	// --- Token Management ---
	RefreshTokens(ctx context.Context, refreshToken string) (*models.TokenResponse, error)
	RevokeAllUserRefreshTokens(ctx context.Context, userID int) error

	// UpdateProfile partially updates the user's profile information (full name and/or phone).
	UpdateProfile(ctx context.Context, userID int, req models.UpdateProfileRequest) error

	// --- Campaings ---
	GetAllCampaigns(ctx context.Context, q models.CampaignListQuery) (*models.CampaignPage, error)
	GetCampaignByID(ctx context.Context, id int) (*models.PublicCampaign, error)
	ProcessDonation(ctx context.Context, donation models.DonationRequest) (*models.LedgerEntry, error)
	GetDonationsByDonorUser(ctx context.Context, userID int) (*models.LedgerEntryPage, error)
	GetDonationsByCampaign(ctx context.Context, campaignID int) (*models.LedgerEntryPage, error)
}

type OAuthProvider interface {
	ProviderName() string
	CallbackPath() string
	OAuth2Config() *oauth2.Config
	GetUser(ctx context.Context, tok *oauth2.Token) (models.OAuthUserInfo, error)
}

// Handler holds all dependencies for the HTTP handler layer.
type Handler struct {
	service        IService
	limiter        Limiter
	middleware     *middlewares.Middleware
	logger         *zerolog.Logger
	cfg            *configs.Config
	oauthProviders []OAuthProvider
}

// NewHandler constructs a Handler with all required dependencies injected.
func NewHandler(
	service IService,
	limiter Limiter,
	middleware *middlewares.Middleware,
	logger *zerolog.Logger,
	cfg *configs.Config,
	oauthProviders ...OAuthProvider,
) *Handler {
	return &Handler{
		service:        service,
		limiter:        limiter,
		middleware:     middleware,
		logger:         logger,
		cfg:            cfg,
		oauthProviders: oauthProviders,
	}
}

// InitRoutes registers all application routes and returns the configured Gin engine.
func (h *Handler) InitRoutes() *gin.Engine {
	router := gin.New()
	router.Use(h.CORSMiddleware(), gin.RecoveryWithWriter(gin.DefaultWriter), h.RequestID(), h.middleware.LoggerMiddleware(), h.middleware.AlertMiddleware())
	router.NoRoute(h.noRoute)

	router.GET("/ping", h.ping)

	v1 := router.Group("/api/v1")
	{
		if h.cfg.App.Env != "production" {
			v1.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
				ginSwagger.URL("/api/v1/docs/swagger.yaml"),
			))
			v1.Static("/docs", "./docs")
		}

		v1.GET("/telegram/panic", func(c *gin.Context) {
			panic("telegram test")
		})
		v1.GET("/telegram/5xx", func(c *gin.Context) {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": "test 500",
				},
			)
		})
		oauth := v1.Group("/oauth")
		{
			for _, oauthProvider := range h.oauthProviders {

				oauthPath := "/" + oauthProvider.ProviderName()
				oauthCallbackPath := oauthPath + "/" + oauthProvider.CallbackPath()

				oauth.GET(oauthPath, h.oauth(oauthProvider.OAuth2Config()))
				oauth.GET(oauthCallbackPath, h.oauthCallback(oauthProvider))
			}
		}

		v1.POST("/send_otp", h.sendOTP)
		v1.POST("/confirm_otp", h.confirmOTP)
		v1.POST("/login", h.login)
		v1.POST("/register", h.register)
		v1.POST("/logout", h.middleware.AuthMiddleware(), h.logout)
		v1.POST("/refresh", h.refreshTokens)

		v1.GET("/check_access", h.middleware.AuthMiddleware(), func(c *gin.Context) {
			h.success(c, "valid")
		})
		v1.GET("/me", h.middleware.AuthMiddleware(), h.getMe)
		userHandler := NewUserHandler(h.service.(*services.Service))
		v1.PATCH("/me", h.middleware.AuthMiddleware(), userHandler.UpdateProfile)
		v1.GET("/stats", h.getStats)
		v1.GET("/sms/balance", h.middleware.AuthMiddleware(), h.getSMSBalance)

		v1.GET("/institutions", h.getAllInstitutions)
		v1.GET("/institutions/:id", h.getInstitutionByID)
		v1.POST("/institutions", h.middleware.AuthMiddleware(models.RoleSuperAdmin), h.createInstitution)

		v1.GET("/institutions/:id/needs", h.getNeedsByInstitution)
		v1.GET("/institutions/:id/needs/:needID", h.getNeedByID)

		// Need management (employees and super-admins only).
		needs := v1.Group("/needs")
		needs.Use(h.middleware.AuthMiddleware(models.RoleEmployee, models.RoleSuperAdmin))
		{
			needs.POST("", h.createNeed)
			needs.PUT("/:id", h.updateNeed)
			needs.DELETE("/:id", h.deleteNeed)
		}

		// Volunteer booking routes (any authenticated user).
		bookings := v1.Group("/bookings")
		bookings.Use(h.middleware.AuthMiddleware())
		{
			bookings.POST("", h.createBooking)
			bookings.GET("/my", h.getMyBookings)
			bookings.PUT("/my/:id/cancel", h.cancelMyBooking)
			bookings.PUT("/my/:id", h.updateMyBooking)
		}

		// Booking management routes (employees and super-admins only).
		bookingMgmt := v1.Group("/bookings")
		bookingMgmt.Use(h.middleware.AuthMiddleware(models.RoleEmployee, models.RoleSuperAdmin))
		{
			bookingMgmt.PUT("/:id/approve", h.approveBooking)
			bookingMgmt.PUT("/:id/reject", h.rejectBooking)
			bookingMgmt.PUT("/:id/complete", h.completeBooking)
		}

		// Institution booking view (employees and super-admins only).
		institutionBookings := v1.Group("/institutions/:id/bookings")
		institutionBookings.Use(h.middleware.AuthMiddleware(models.RoleEmployee, models.RoleSuperAdmin))
		{
			institutionBookings.GET("", h.getInstitutionBookings)
		}

		// Event routes.
		v1.GET("/events", h.middleware.OptionalAccessToken(), h.getAllEvents)
		v1.GET("/events/:id", h.middleware.OptionalAccessToken(), h.getEventByID)
		v1.POST("/events", h.middleware.AuthMiddleware(models.RoleEmployee, models.RoleSuperAdmin), h.createEvent)
		v1.POST("/events/:id/join", h.middleware.AuthMiddleware(), h.joinEvent)
		v1.DELETE("/events/:id/leave", h.middleware.AuthMiddleware(), h.leaveEvent)

		v1.GET("/institutions/:id/events", h.middleware.AuthMiddleware(models.RoleEmployee, models.RoleSuperAdmin), h.getInstitutionEvents)

		eventMgmt := v1.Group("/events")
		eventMgmt.Use(h.middleware.AuthMiddleware(models.RoleEmployee, models.RoleSuperAdmin))
		{
			eventMgmt.PUT("/:id/approve", h.approveEvent)
			eventMgmt.PUT("/:id/reject", h.rejectEvent)
		}

		// Vacancies and team members (public).
		v1.GET("/vacancies", h.getAllVacancies)
		v1.GET("/vacancies/:id", h.getVacancyByID)

		v1.GET("/team", h.getAllTeamMembers)
		v1.GET("/team/:id", h.getTeamMemberByID)

		// Campaigns.
		v1.GET("/campaigns", h.getActiveCampaigns)
		v1.GET("/campaigns/:id", h.getCampaignByID)
		v1.GET("/campaigns/:id/ledger", h.getCampaignLedger)
		v1.POST("/campaigns/:id/donate", h.middleware.AuthMiddleware(), h.donate)
		v1.GET("/donations/my", h.middleware.AuthMiddleware(), h.getMyDonations)
	}
	return router
}

func (h *Handler) ping(context *gin.Context) {
	h.respond(context, "pong", http.StatusOK)
}

func (h *Handler) noRoute(context *gin.Context) {
	h.respond(context, "this route is not supported", http.StatusNotFound)
}

func (h *Handler) respond(context *gin.Context, obj interface{}, code int) {
	context.JSON(code, obj)
}

func (h *Handler) success(c *gin.Context, data any) {
	h.respond(c, models.Response{
		Message: "Success",
		Data:    data,
	}, http.StatusOK)
}

// handleError maps domain errors to the appropriate HTTP status codes and
// response bodies.
func (h *Handler) handleError(c *gin.Context, err error) {
	log := zerolog.Ctx(c.Request.Context())

	badReq := &myerrors.BadRequestErr{}
	forbidden := &myerrors.ForbiddenErr{}
	unprocessable := &myerrors.UnprocessableErr{}
	unauth := &myerrors.UnauthorizedErr{}
	manyReq := &myerrors.TooManyRequestsErr{}
	conflict := &myerrors.ConflictErr{}

	switch {
	case errors.Is(err, myerrors.ErrNotFound):
		log.Warn().Err(err).Msg("not found")
		c.JSON(http.StatusNotFound, gin.H{"message": myerrors.ErrNotFound.Error()})
	case errors.As(err, unprocessable):
		log.Warn().Err(err).Msg("unprocessable entity")
		c.JSON(http.StatusUnprocessableEntity, unprocessable)
	case errors.As(err, badReq):
		log.Warn().Err(err).Msg("bad request")
		c.JSON(http.StatusBadRequest, badReq)
	case errors.As(err, forbidden):
		log.Warn().Err(err).Msg("forbidden")
		c.JSON(http.StatusForbidden, forbidden)
	case errors.As(err, unauth):
		log.Warn().Err(err).Msg("unauthorized")
		c.JSON(http.StatusUnauthorized, unauth)
	case errors.As(err, manyReq):
		log.Warn().Err(err).Msg("too many requests")
		c.JSON(http.StatusTooManyRequests, manyReq)
	case errors.As(err, conflict):
		log.Warn().Err(err).Msg("conflict")
		c.JSON(http.StatusConflict, conflict)
	default:
		log.Error().Err(err).Msg("internal server error")
		c.JSON(http.StatusInternalServerError, myerrors.InternalError())
	}
	c.Abort()
}

// RequestID is a middleware that ensures every request carries a unique
// request ID header and propagates it through the request context.
func (h *Handler) RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.Request.Header.Get(constants.RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}
		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, constants.RequestIDKey, requestID)
		log := h.logger.With().Str("request_id", requestID).Logger()
		ctx = log.WithContext(ctx)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// CORSMiddleware sets permissive CORS headers for recognised origins and
// handles pre-flight OPTIONS requests.
func (h *Handler) CORSMiddleware() gin.HandlerFunc {
	allowedOrigins := map[string]bool{
		"https://hadaf.tj":      true,
		"https://www.hadaf.tj":  true,
		"http://localhost:3000": true,
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if allowedOrigins[origin] {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, x-request-id")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func (h *Handler) mustGetUserID(c *gin.Context) (int, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		h.handleError(c, myerrors.NewUnauthorizedErr("user not authenticated"))
		return 0, true
	}

	userIDInt, ok := userID.(int)
	if !ok {
		h.handleError(c, myerrors.NewUnauthorizedErr("invalid user ID"))
		return 0, true
	}

	return userIDInt, false
}
