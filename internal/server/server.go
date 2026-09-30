package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"html/template"
	"io/fs"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/cache"
	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/database"
	authHandlerPkg "github.com/Varunjp/vyavsa/internal/handler/auth"
	"github.com/Varunjp/vyavsa/internal/handler/health"
	platformHandlerPkg "github.com/Varunjp/vyavsa/internal/handler/platform"
	tenantHandlerPkg "github.com/Varunjp/vyavsa/internal/handler/tenant"
	webHandlerPkg "github.com/Varunjp/vyavsa/internal/handler/web"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/mailer"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/Varunjp/vyavsa/internal/middleware"
	"github.com/Varunjp/vyavsa/internal/repository"
	postgresRepo "github.com/Varunjp/vyavsa/internal/repository/postgres"
	redisRepo "github.com/Varunjp/vyavsa/internal/repository/redis"
	"github.com/Varunjp/vyavsa/internal/service"
	"github.com/Varunjp/vyavsa/pkg/response"
	"github.com/Varunjp/vyavsa/web"
	"github.com/gin-gonic/gin"
)

// Server coordinates the HTTP server, routing, dependencies, and lifecycle
type Server struct {
	cfg     *config.Config
	log     *logger.Logger
	db      *database.Postgres
	redis   *cache.Redis
	metrics *metrics.Metrics
	router  *gin.Engine
	httpSrv *http.Server
}

// New creates and configures a new Server instance
func New(
	cfg *config.Config,
	log *logger.Logger,
	db *database.Postgres,
	redis *cache.Redis,
	m *metrics.Metrics,
) *Server {
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	s := &Server{
		cfg:     cfg,
		log:     log,
		db:      db,
		redis:   redis,
		metrics: m,
		router:  router,
	}

	s.setupMiddlewares()
	s.setupRoutes()

	s.httpSrv = &http.Server{
		Addr:              ":" + cfg.App.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return s
}

// Router returns the configured Gin engine (useful for handler and router testing)
func (s *Server) Router() *gin.Engine {
	return s.router
}

func (s *Server) setupMiddlewares() {
	s.router.Use(
		middleware.RequestID(),
		middleware.Recovery(s.log),
		middleware.SecurityHeaders(s.cfg.App.Env == "production"),
		middleware.CORS(s.cfg.CORS.AllowedOrigins),
		middleware.RequestLogger(s.log, s.metrics),
	)
}

func (s *Server) setupRoutes() {
	// Setup user-facing landing page, auth routes, and embedded static assets
	s.setupWebRoutes()

	healthHandler := health.NewHandler(s.db, s.redis)

	// Liveness check (process is up)
	s.router.GET("/health", healthHandler.Health)

	// Readiness check (dependencies available)
	s.router.GET("/ready", healthHandler.Ready)

	// Prometheus metrics endpoint
	if s.cfg.Metrics.Enabled && s.metrics != nil {
		s.router.GET(s.cfg.Metrics.Path, gin.WrapH(s.metrics.Handler()))
	}

	// Base API v1 group
	apiV1 := s.router.Group("/api/v1")
	{
		// Ping / heartbeat inside API v1
		apiV1.GET("/ping", func(c *gin.Context) {
			response.Success(c, gin.H{"pong": true}, "API v1 is active")
		})

		// Wire Core Feature Routes (Auth, Platform Admin, Tenant Onboarding & Management)
		s.setupAPIRoutes(apiV1)
	}

	// 404 handler returning standard JSON error
	s.router.NoRoute(func(c *gin.Context) {
		response.CustomError(c, http.StatusNotFound, "ROUTE_NOT_FOUND", fmt.Sprintf("path '%s' not found", c.Request.URL.Path))
	})
}

func (s *Server) setupWebRoutes() {
	// Parse HTML templates from embedded filesystem
	tmpl, err := template.ParseFS(web.WebFS, "templates/*.html")
	if err != nil {
		s.log.Error("failed to parse web templates", "error", err)
	} else {
		s.router.SetHTMLTemplate(tmpl)
	}

	// Serve static assets from embedded filesystem
	staticFS, err := fs.Sub(web.WebFS, "static")
	if err != nil {
		s.log.Error("failed to create static sub-filesystem", "error", err)
	} else {
		s.router.StaticFS("/static", http.FS(staticFS))
	}

	webHandler := webHandlerPkg.NewHandler()

	// Public Web Pages
	s.router.GET("/", webHandler.ShowLanding)
	s.router.GET("/login", webHandler.ShowLogin)
	s.router.GET("/register", webHandler.ShowRegister)
	s.router.GET("/verify-otp", webHandler.ShowVerifyOTP)
	s.router.GET("/forgot-password", webHandler.ShowForgotPassword)
	s.router.GET("/reset-password", webHandler.ShowResetPassword)
	s.router.GET("/platform/login", webHandler.ShowPlatformLogin)
	s.router.GET("/dashboard", webHandler.ShowDashboard)

	// Platform Admin Web Views
	s.router.GET("/platform", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/platform/dashboard")
	})
	s.router.GET("/platform/dashboard", webHandler.ShowPlatformDashboard)
	s.router.GET("/platform/tenants", webHandler.ShowPlatformDashboard)
	s.router.GET("/platform/tenants/:id", webHandler.ShowPlatformDashboard)
	s.router.GET("/platform/plans", webHandler.ShowPlatformDashboard)
	s.router.GET("/platform/subscriptions", webHandler.ShowPlatformDashboard)
	s.router.GET("/platform/transactions", webHandler.ShowPlatformDashboard)
	s.router.GET("/platform/metrics", webHandler.ShowPlatformDashboard)

	// Favicon shortcut
	s.router.GET("/favicon.ico", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/static/images/favicon.svg")
	})

}

func (s *Server) setupAPIRoutes(apiV1 *gin.RouterGroup) {
	jwtManager := auth.NewJWTManager(s.cfg.JWT)
	hasher := auth.NewBcryptHasher()

	// Repositories
	var platformAdminRepo repository.PlatformAdminRepository
	var tenantUserRepo repository.TenantUserRepository
	var planRepo repository.PlatformPlanRepository
	var tenantRepo repository.TenantRepository
	var summaryRepo repository.TenantFinancialSummaryRepository
	var subRepo repository.PlatformSubscriptionRepository
	var txnRepo repository.PlatformPlanTransactionRepository
	var transactor repository.Transactor

	var empRepo repository.TenantEmployeeRepository
	var custRepo repository.TenantCustomerRepository
	var bankRepo repository.TenantBankRepository
	var lineSaleRepo repository.LineSaleRepository
	var countSaleRepo repository.CounterSaleRepository
	var purchRepo repository.TenantPurchaseRepository
	var expRepo repository.TenantExpenseRepository
	var attRepo repository.AttendanceRepository
	var salaryRepo repository.EmployeeSalaryRepository
	var statsRepo repository.TenantDailyStatsRepository

	if s.db != nil && s.db.Pool != nil {
		platformAdminRepo = postgresRepo.NewPlatformAdminPostgres(s.db.Pool)
		tenantUserRepo = postgresRepo.NewTenantUserPostgres(s.db.Pool)
		planRepo = postgresRepo.NewPlatformPlanPostgres(s.db.Pool)
		tenantRepo = postgresRepo.NewTenantPostgres(s.db.Pool)
		summaryRepo = postgresRepo.NewTenantFinancialSummaryPostgres(s.db.Pool)
		subRepo = postgresRepo.NewPlatformSubscriptionPostgres(s.db.Pool)
		txnRepo = postgresRepo.NewPlatformPlanTransactionPostgres(s.db.Pool)
		transactor = postgresRepo.NewPostgresTransactor(s.db.Pool)

		empRepo = postgresRepo.NewTenantEmployeePostgres(s.db.Pool)
		custRepo = postgresRepo.NewTenantCustomerPostgres(s.db.Pool)
		bankRepo = postgresRepo.NewTenantBankPostgres(s.db.Pool)
		lineSaleRepo = postgresRepo.NewLineSalePostgres(s.db.Pool)
		countSaleRepo = postgresRepo.NewCounterSalePostgres(s.db.Pool)
		purchRepo = postgresRepo.NewTenantPurchasePostgres(s.db.Pool)
		expRepo = postgresRepo.NewTenantExpensePostgres(s.db.Pool)
		attRepo = postgresRepo.NewAttendancePostgres(s.db.Pool)
		salaryRepo = postgresRepo.NewEmployeeSalaryPostgres(s.db.Pool)
		statsRepo = postgresRepo.NewTenantDailyStatsPostgres(s.db.Pool)
	}

	var blacklistRepo repository.TokenBlacklistRepository
	var passwordResetRepo repository.PasswordResetRepository
	if s.redis != nil {
		blacklistRepo = redisRepo.NewTokenBlacklistRedis(s.redis)
		passwordResetRepo = redisRepo.NewPasswordResetRedis(s.redis)
	}

	appMailer := mailer.NewMailer(s.cfg.Mailer, s.log.Logger)

	// Services
	authService := service.NewAuthService(
		platformAdminRepo,
		tenantUserRepo,
		blacklistRepo,
		passwordResetRepo,
		appMailer,
		hasher,
		jwtManager,
		s.cfg.PasswordReset,
		s.metrics,
		s.log.Logger,
	)

	planService := service.NewPlatformPlanService(
		planRepo,
		s.log.Logger,
	)

	tenantService := service.NewTenantService(
		tenantRepo,
		tenantUserRepo,
		summaryRepo,
		subRepo,
		planRepo,
		txnRepo,
		transactor,
		hasher,
		jwtManager,
		s.metrics,
		s.log.Logger,
	)

	opsService := service.NewTenantOperationsService(
		tenantUserRepo,
		empRepo,
		custRepo,
		bankRepo,
		lineSaleRepo,
		countSaleRepo,
		purchRepo,
		expRepo,
		attRepo,
		salaryRepo,
		statsRepo,
		summaryRepo,
		transactor,
		hasher,
		s.log.Logger,
	)
	if s.cfg.App.Location != nil {
		opsService.SetLocation(s.cfg.App.Location)
	}

	// Handlers
	authHandler := authHandlerPkg.NewHandler(authService)
	planHandler := platformHandlerPkg.NewPlanHandler(planService)
	platformTenantHandler := platformHandlerPkg.NewTenantHandler(tenantService)
	tenantHandler := tenantHandlerPkg.NewTenantHandler(tenantService)
	opsHandler := tenantHandlerPkg.NewOperationsHandler(opsService)

	// 1. Public Endpoints
	authGroup := apiV1.Group("/auth")
	{
		authGroup.POST("/platform/login", authHandler.PlatformLogin)
		authGroup.POST("/tenant/login", authHandler.TenantLogin)
		authGroup.POST("/tenant/register", tenantHandler.Register)
		authGroup.POST("/refresh", authHandler.RefreshToken)
		authGroup.POST("/forgot-password", authHandler.ForgotPassword)
		authGroup.POST("/verify-reset-otp", authHandler.VerifyResetOTP)
		authGroup.POST("/reset-password", authHandler.ResetPassword)
	}

	// Also support root /auth paths directly
	rootAuth := s.router.Group("/auth")
	{
		rootAuth.POST("/forgot-password", authHandler.ForgotPassword)
		rootAuth.POST("/verify-reset-otp", authHandler.VerifyResetOTP)
		rootAuth.POST("/reset-password", authHandler.ResetPassword)
	}

	// Public Tenant Self-Registration (also accessible under /api/v1/tenants/register)
	apiV1.POST("/tenants/register", tenantHandler.Register)

	// Public Subscription Plans Catalog (viewable without authentication)
	apiV1.GET("/plans", planHandler.List)

	// 2. Protected Endpoints (Requires valid JWT)
	protected := apiV1.Group("")
	protected.Use(middleware.Authenticate(jwtManager, blacklistRepo))
	{
		protected.POST("/auth/logout", authHandler.Logout)
		protected.GET("/auth/me", authHandler.GetMe)

		// ----------------------------------------------------
		// Platform Administrator Gated Routes
		// ----------------------------------------------------
		platform := protected.Group("/platform")
		platform.Use(middleware.RequirePlatformAdmin())
		{
			platform.GET("/ping", func(c *gin.Context) {
				claims, _ := auth.GetClaims(c)
				response.Success(c, gin.H{
					"admin_id": claims.UserID,
					"email":    claims.Email,
					"role":     claims.Role,
				}, "platform admin authenticated")
			})

			// Subscription Plan Management
			platform.POST("/plans", planHandler.Create)
			platform.GET("/plans", planHandler.List)
			platform.GET("/plans/:id", planHandler.GetByID)
			platform.PUT("/plans/:id", planHandler.Update)
			platform.DELETE("/plans/:id", planHandler.Archive)

			// Tenant Onboarding & Management
			platform.POST("/tenants", platformTenantHandler.Onboard)
			platform.GET("/tenants", platformTenantHandler.List)
			platform.GET("/tenants/:id", platformTenantHandler.GetByID)
			platform.PUT("/tenants/:id", platformTenantHandler.UpdateTenant)
			platform.PATCH("/tenants/:id/status", platformTenantHandler.UpdateStatus)
			platform.POST("/tenants/:id/subscription", platformTenantHandler.ChangeSubscription)

			// Platform Dashboard, Subscriptions & Transactions
			platform.GET("/dashboard/metrics", platformTenantHandler.GetDashboardMetrics)
			platform.GET("/subscriptions", platformTenantHandler.ListSubscriptions)
			platform.GET("/transactions", platformTenantHandler.ListTransactions)
		}

		// ----------------------------------------------------
		// Tenant Member Gated Routes
		// ----------------------------------------------------
		tenant := protected.Group("/tenant")
		tenant.Use(middleware.RequireTenantUser())
		{
			tenant.GET("/ping", func(c *gin.Context) {
				claims, _ := auth.GetClaims(c)
				response.Success(c, gin.H{
					"user_id":   claims.UserID,
					"tenant_id": claims.TenantID,
					"role":      claims.Role,
				}, "tenant user authenticated")
			})

			// Tenant Organization Profile & Financial Insights
			tenant.GET("/profile", tenantHandler.GetProfile)
			tenant.GET("/settings", tenantHandler.GetProfile)
			tenant.GET("/financial-summary", tenantHandler.GetFinancialSummary)
			tenant.GET("/subscription", tenantHandler.GetSubscription)
			tenant.GET("/subscription/plans", tenantHandler.ListPlans)
			tenant.GET("/plans", tenantHandler.ListPlans)

			// Line Sales (Admin & Tenant User)
			tenant.POST("/line-sales", opsHandler.CreateLineSale)
			tenant.GET("/line-sales", opsHandler.ListLineSales)
			tenant.GET("/line-sales/:id", opsHandler.GetLineSaleByID)
			tenant.PUT("/line-sales/:id", opsHandler.UpdateLineSale)
			tenant.DELETE("/line-sales/:id", opsHandler.DeleteLineSale)

			// Counter Sales (Admin & Tenant User)
			tenant.POST("/counter-sales", opsHandler.CreateCounterSale)
			tenant.GET("/counter-sales", opsHandler.ListCounterSales)
			tenant.GET("/counter-sales/:id", opsHandler.GetCounterSaleByID)
			tenant.PUT("/counter-sales/:id", opsHandler.UpdateCounterSale)
			tenant.DELETE("/counter-sales/:id", opsHandler.DeleteCounterSale)

			// Purchases (Admin & Tenant User)
			tenant.POST("/purchases", opsHandler.CreatePurchase)
			tenant.GET("/purchases", opsHandler.ListPurchases)
			tenant.GET("/purchases/:id", opsHandler.GetPurchaseByID)
			tenant.PUT("/purchases/:id", opsHandler.UpdatePurchase)
			tenant.DELETE("/purchases/:id", opsHandler.DeletePurchase)

			// Expenses (Admin & Tenant User)
			tenant.POST("/expenses", opsHandler.CreateExpense)
			tenant.GET("/expenses", opsHandler.ListExpenses)
			tenant.GET("/expenses/:id", opsHandler.GetExpenseByID)
			tenant.PUT("/expenses/:id", opsHandler.UpdateExpense)
			tenant.DELETE("/expenses/:id", opsHandler.DeleteExpense)

			// Attendance, OT, Advances (Admin & Tenant User)
			tenant.POST("/attendance", opsHandler.RecordAttendance)
			tenant.GET("/attendance", opsHandler.ListAttendance)
			tenant.GET("/attendance/:id", opsHandler.GetAttendanceByID)
			tenant.PUT("/attendance/:id", opsHandler.UpdateAttendance)
			tenant.POST("/overtime", opsHandler.RecordOvertime)
			tenant.POST("/advances", opsHandler.RecordAdvance)

			// Daily Stats & Dashboard Summary
			tenant.GET("/daily-stats", opsHandler.GetDailyStats)
			tenant.GET("/dashboard", opsHandler.GetFinancialMetrics)
			tenant.GET("/dashboard/today", opsHandler.GetTodayOverview)
			tenant.GET("/dashboard/overview", opsHandler.GetTodayOverview)
			tenant.GET("/metrics", opsHandler.GetFinancialMetrics)

			// ----------------------------------------------------
			// Tenant Administrator Only Routes
			// ----------------------------------------------------
			tenantAdmin := tenant.Group("")
			tenantAdmin.Use(middleware.RequireRole(auth.RoleTenantAdmin))
			{
				// Tenant Organization Settings
				tenantAdmin.PUT("/settings", tenantHandler.UpdateSettings)
				tenantAdmin.PUT("/profile", tenantHandler.UpdateSettings)

				// Subscription & Transactions Management
				tenantAdmin.POST("/subscription/purchase", tenantHandler.PurchasePlan)
				tenantAdmin.POST("/subscription/upgrade", tenantHandler.PurchasePlan)
				tenantAdmin.GET("/transactions", tenantHandler.ListTransactions)
				tenantAdmin.GET("/subscription/transactions", tenantHandler.ListTransactions)

				// Tenant Users Management
				tenantAdmin.POST("/users", opsHandler.CreateTenantUser)
				tenantAdmin.GET("/users", opsHandler.ListTenantUsers)
				tenantAdmin.GET("/users/:id", opsHandler.GetTenantUserByID)
				tenantAdmin.PUT("/users/:id", opsHandler.UpdateTenantUser)
				tenantAdmin.DELETE("/users/:id", opsHandler.DeleteTenantUser)

				// Employees Management
				tenantAdmin.POST("/employees", opsHandler.CreateEmployee)
				tenantAdmin.GET("/employees", opsHandler.ListEmployees)
				tenantAdmin.GET("/employees/:id", opsHandler.GetEmployeeByID)
				tenantAdmin.PUT("/employees/:id", opsHandler.UpdateEmployee)
				tenantAdmin.DELETE("/employees/:id", opsHandler.DeleteEmployee)

				// Customers Management
				tenantAdmin.POST("/customers", opsHandler.CreateCustomer)
				tenantAdmin.GET("/customers", opsHandler.ListCustomers)
				tenantAdmin.GET("/customers/:id", opsHandler.GetCustomerByID)
				tenantAdmin.GET("/customers/:id/adjustments", opsHandler.ListCustomerAdjustments)
				tenantAdmin.PUT("/customers/:id", opsHandler.UpdateCustomer)
				tenantAdmin.POST("/customers/:id/adjust-balance", opsHandler.AdjustCustomerBalance)
				tenantAdmin.DELETE("/customers/:id", opsHandler.DeleteCustomer)

				// Banks Management
				tenantAdmin.POST("/banks", opsHandler.CreateBank)
				tenantAdmin.GET("/banks", opsHandler.ListBanks)
				tenantAdmin.GET("/banks/:id", opsHandler.GetBankByID)
				tenantAdmin.GET("/banks/:id/transactions", opsHandler.ListBankTransactions)
				tenantAdmin.PUT("/banks/:id", opsHandler.UpdateBank)
				tenantAdmin.DELETE("/banks/:id", opsHandler.DeleteBank)

				// Employee Salaries Management
				tenantAdmin.GET("/salaries", opsHandler.ListSalaries)
				tenantAdmin.GET("/salaries/pending", opsHandler.ListPendingSalaries)
				tenantAdmin.GET("/salaries/:employee_id", opsHandler.GetSalaryByEmployeeID)
				tenantAdmin.PUT("/salaries/:employee_id", opsHandler.UpdateSalaryBalance)
				tenantAdmin.POST("/salaries/:employee_id/pay", opsHandler.PaySalary)
				tenantAdmin.GET("/salaries/:employee_id/payments", opsHandler.ListSalaryPayments)
				tenantAdmin.GET("/salary-payments", opsHandler.ListSalaryPayments)
			}
		}
	}
}

// Run starts the HTTP server and blocks until an interrupt signal is received for graceful shutdown
func (s *Server) Run() error {
	shutdownErr := make(chan error, 1)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		sig := <-quit

		s.log.Info("shutdown signal received", slog.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), s.cfg.App.ShutdownTimeout)
		defer cancel()

		var errs []error

		// Step 1: Stop accepting new HTTP requests and finish in-flight requests
		s.log.Info("stopping HTTP server")
		if err := s.httpSrv.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("http server shutdown: %w", err))
		}

		// Step 2: Close Redis client
		if s.redis != nil {
			if err := s.redis.Close(); err != nil {
				errs = append(errs, fmt.Errorf("redis close: %w", err))
			}
		}

		// Step 3: Close Postgres connection pool
		if s.db != nil {
			s.db.Close()
		}

		s.log.Info("all components stopped gracefully")

		if len(errs) > 0 {
			shutdownErr <- errors.Join(errs...)
		} else {
			shutdownErr <- nil
		}
	}()

	s.log.Info("starting HTTP server",
		slog.String("port", s.cfg.App.Port),
		slog.String("env", s.cfg.App.Env),
		slog.String("app_name", s.cfg.App.Name),
	)

	if err := s.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server failed: %w", err)
	}

	return <-shutdownErr
}
