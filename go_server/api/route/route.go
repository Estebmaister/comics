package route

import (
	"net/http"

	"comics/api/controller"
	"comics/api/middleware"
	"comics/bootstrap"
	"comics/docs"
	"comics/domain"
	"comics/internal/health"
	"comics/internal/service"
	"comics/internal/tokenutil"

	"github.com/gin-gonic/gin"
	"github.com/penglongli/gin-metrics/ginmetrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

const (
	// Tracing
	tracingServiceName = "comics_router"

	// Headers
	keyRole          = middleware.KeyRole
	keyAuthorization = middleware.KeyAuthorization
	keyAccept        = middleware.KeyAccept
	contentTypeJSON  = middleware.ContentTypeJSON

	// Cookies
	cookieAccessToken  = middleware.KeyAccessToken
	cookieRefreshToken = middleware.KeyRefreshToken
)

// Setup configures the gin routes of the server
func Setup(env *bootstrap.Env, userRepo domain.UserStore, g *gin.Engine) {

	// Starting user service and inject it into auth controller
	userService := service.NewUserService(userRepo, env)
	authController := controller.NewAuthControl(userService, env)

	// get global Monitor object
	m := ginmetrics.GetMonitor()
	// m.SetMetricPath("/debug/metrics") // TODO: extract metrics to middlewares
	m.Use(g) // set middleware for gin
	g.Use(   // adds HTTP tracer instrumentation and request logging for the whole router
		otelgin.Middleware(tracingServiceName),
		middleware.LoggerMiddleware(),
	)

	g.Static("/static", "./static") // Serve static files (CSS, JS, images)
	g.LoadHTMLGlob("templates/*")   // Load templates from "templates/" directory

	// Redirect root to protected profile
	g.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusSeeOther, "/protected/profile")
	})

	basePath := "/"
	publicRouter := g.Group(basePath)
	setOAuth2(env, authController, publicRouter)
	{ // All Public APIs
		swaggerRouter(env, basePath, publicRouter)
		metricsRouter(userRepo, publicRouter)
		signUpRouter(authController, publicRouter)
		loginRouter(authController, publicRouter)
		refreshTokenRouter(authController, publicRouter)
	}

	protected := g.Group("/protected")
	protected.Use( // verifies AccessToken
		middleware.AuthenticationMiddleware(env.JWTConfig.AccessTokenSecret))
	{ // All protected Private APIs
		protected.GET("/profile", getProfile(authController))
		protected.PUT("/profile", putProfile(authController))
		NewTaskRouter(env, userRepo, protected)
	}

	admin := g.Group("/admin")
	admin.Use( // verifies AccessToken and Role
		middleware.AuthenticationMiddleware(env.JWTConfig.AccessTokenSecret),
		middleware.RoleMiddleware(tokenutil.RoleAdmin))
	{ // All admin APIs
		dashboardRouter(userRepo, admin)
	}
}

// metricsRouter manages the Prometheus metrics
//
//	@Summary		Metrics
//	@Description	Returns metrics necessary for observability
//	@ID				metrics
//	@Tags			Metrics
//	@Accept			json
//	@Produce		json
//	@Success		200	string	string	"Metrics: \#TYPE & \#HELP"
//	@Failure		503	string	string	"Service unavailable"
//	@Router			/metrics [get]
func metricsRouter(userRepo domain.UserStore, group *gin.RouterGroup) {
	prometheus.MustRegister(collectors.NewBuildInfoCollector())
	// prometheus.MustRegister(collectors.NewGoCollector())
	group.GET("/metrics", gin.WrapH(promhttp.Handler()))
	healthy := health.NewHealthChecker(userRepo)

	// Register health check handlers
	group.GET("/health", gin.WrapH(healthy.LivenessHandler()))
	group.GET("/ready", gin.WrapH(healthy.ReadinessHandler()))

	// Start health checker
	healthy.Start()
}

func swaggerRouter(env *bootstrap.Env, basePath string, group *gin.RouterGroup) {
	// Setting runtine values in SwaggerInfo
	docs.SwaggerInfo.BasePath = basePath
	docs.SwaggerInfo.Host = env.HostURL
	docs.SwaggerInfo.Schemes = []string{"http", "https"}
	log.Info().
		Str("URL", "https://"+env.AddressHTTP+":"+env.PortHTTP+"/swagger/index.html").
		Msg("Swagger")

	// Swagger API documentation
	group.GET("/swagger/*any",
		ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// dashboardRouter returns a dashboard view for admins
//
//	@Summary		Dashboard
//	@Description	Returns the admin dashboard, needs admin auth
//	@ID				dashboard
//	@Tags			Dashboard
//	@Security		Bearer JWT
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string				true	"Bearer JWT"	default(Bearer XXX)
//	@Success		200				{object}	map[string]string	"OK"
//	@Failure		400				{string}	string				"Not registered"
//	@Failure		404				{string}	string				"Not implemented"
//	@Router			/admin/dashboard [get]
func dashboardRouter(userRepo domain.UserStore, group *gin.RouterGroup) {
	group.GET("/dashboard", func(c *gin.Context) {
		// Fetch metrics from repository
		metrics := userRepo.GetStats()

		if c.GetHeader(keyAccept) == contentTypeJSON {
			c.JSON(http.StatusOK, metrics)
			return
		}
		// Render HTML template with metrics data
		otelgin.HTML(c, http.StatusOK, "dashboard.html", gin.H{
			"Title":   "Database Metrics Dashboard",
			"Metrics": metrics,
		})
	})
}

// signUpRouter tries to create a new user from the body provided
//
//	@Summary		SignUp new user
//	@Description	Signs Up a new user (for demonstration purposes),
//	@Description	receive a confirmation for success or failure
//	@ID				user-signup
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			user	body		domain.SignUpRequest	true	"Login user"
//	@Success		201		{object}	domain.AuthResponse		"registered"
//	@Failure		400		{object}	domain.NoDataResponse	"not registered, invalid data"
//	@Failure		409		{object}	domain.NoDataResponse	"username or email already in use"
//	@Router			/signup [post]
func signUpRouter(authController *controller.AuthControl, group *gin.RouterGroup) {
	group.GET("/signup", func(c *gin.Context) {
		otelgin.HTML(c, http.StatusOK, "signup.html", nil)
	})

	group.POST("/signup", func(c *gin.Context) {
		var user domain.SignUpRequest

		// Check user input
		if err := c.ShouldBindJSON(&user); err != nil {
			c.Error(err) // nolint:errcheck
			c.JSON(http.StatusBadRequest, &domain.APIResponse[any]{
				Status: http.StatusBadRequest, Message: "Invalid data, imposible to parse"})
			return
		}

		resp, err := authController.Register(c.Request.Context(), user)
		if err != nil {
			c.Error(err) // nolint:errcheck
		}
		c.JSON(resp.Status, resp)
	})
}

// loginRouter log a user or admin and creates token credentials
//
//	@Summary		Login existent user
//	@ModuleID		signUp
//	@Description	Login a user with basic credentials to receive an auth 'token'
//	@Description	in the headers if successful
//	@Security		Bearer JWT
//	@ID				user-login
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string				false	"Bearer JWT"	default(Bearer XXX)
//	@Param			user			body		domain.LoginRequest	true	"Login user"
//	@Success		200				{object}	domain.AuthResponse	"logged in"
//	@Header			200				{string}	Authorization		"Bearer JWT"
//	@Failure		400				{string}	string				"no ok"
//	@Router			/login [post]
func loginRouter(authController *controller.AuthControl, group *gin.RouterGroup) {
	group.GET("/login", func(c *gin.Context) {
		otelgin.HTML(c, http.StatusOK, "login.html", nil)
	})

	group.POST("/login", func(c *gin.Context) {
		var user domain.LoginRequest
		accessToken := c.GetHeader(keyAuthorization)

		// Check user input
		if err := c.ShouldBindJSON(&user); err != nil && accessToken == "" {
			c.Error(err) // nolint:errcheck
			c.JSON(http.StatusBadRequest, &domain.APIResponse[any]{
				Status: http.StatusBadRequest, Message: "Invalid data, imposible to parse"})
			return
		}

		resp, err := authController.Login(c.Request.Context(), accessToken, user)
		if err != nil {
			c.Error(err) // nolint:errcheck
			c.JSON(resp.Status, resp)
			return
		}
		c.Header(keyAuthorization, "Bearer "+resp.Data.AccessToken)
		// Set JWT in HttpOnly cookies (for web clients)
		c.SetCookie(cookieAccessToken, resp.Data.AccessToken,
			authController.GetAccessTokenExpirySeconds(), "/",
			"", true, true)
		c.SetCookie(cookieRefreshToken, resp.Data.RefreshToken,
			authController.GetRefreshTokenExpirySeconds(), "/",
			"", true, true)
		c.JSON(resp.Status, resp)
	})
}

// refreshTokenRouter tries to renovate the credentials of a logged user
//
//	@Summary		RefreshToken
//	@Description	Function for refreshing the access token
//	@ID				refresh-token
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string				true	"Bearer JWT"	default(Bearer refresh_token)
//	@Param			Role			header		string				false	"role"			Enums(user, admin)
//	@Success		200				{object}	domain.AuthResponse	"new access token generated"
//	@Header			200				{string}	Authorization		"Bearer JWT"
//	@Failure		400				{integer}	string				"not registered"
//	@Failure		404				{string}	integer				"not registered"
//	@Router			/refresh-token [post]
func refreshTokenRouter(authController *controller.AuthControl, group *gin.RouterGroup) {
	group.POST("/refresh-token", func(c *gin.Context) {
		refreshToken := c.GetHeader(keyAuthorization)
		role := c.GetHeader(keyRole)
		if role == "" {
			role = tokenutil.RoleUser
		}

		resp, err := authController.RefreshToken(c.Request.Context(), refreshToken, role)
		if err != nil {
			c.Error(err) // nolint:errcheck
			c.JSON(resp.Status, resp)
			return
		}
		c.Header(keyAuthorization, "Bearer "+resp.Data.AccessToken)
		c.JSON(resp.Status, resp)
	})
}

// NewTaskRouter creates a new router for tasks TODO: implement
func NewTaskRouter(_ *bootstrap.Env, _ domain.UserStore, group *gin.RouterGroup) {
	group.GET("/tasks", func(_ *gin.Context) {
		// Task handler logic
	})
	group.POST("/tasks", func(_ *gin.Context) {
		// Create task handler logic
	})
}
