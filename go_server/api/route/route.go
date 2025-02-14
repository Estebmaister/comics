package route

import (
	"net/http"
	"time"

	"comics/api/controller"
	"comics/api/middleware"
	"comics/bootstrap"
	"comics/docs"
	"comics/domain"
	"comics/internal/repo/mongo"
	"comics/internal/service"
	"comics/internal/tokenutil"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

const (
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
func Setup(
	env *bootstrap.Env, timeout time.Duration, userRepo domain.UserStore, gin *gin.Engine) {
	basePath := "/"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}
	docs.SwaggerInfo.Host = env.ServerAddress
	docs.SwaggerInfo.BasePath = basePath
	println("http://" + env.ServerAddress + "/swagger/index.html")

	userService := service.NewUserService(userRepo, env)
	authController := controller.NewAuthControl(userService, env)

	// Serve static files (CSS, JS, images)
	gin.Static("/static", "./static")
	publicRouter := gin.Group(basePath)
	// All Public APIs
	{
		swaggerRouter(publicRouter)
		signUpRouter(authController, publicRouter)
		loginRouter(authController, publicRouter)
		refreshTokenRouter(authController, publicRouter)
	}

	protectedRouter := gin.Group("/protected")
	// Middleware to verify AccessToken
	protectedRouter.Use(middleware.AuthenticationMiddleware(env.JWT.AccessTokenSecret))
	// All protected Private APIs
	{
		profileRouter(authController, protectedRouter)
		NewTaskRouter(env, timeout, userRepo, protectedRouter)
	}

	// Load templates from "templates/" directory
	gin.LoadHTMLGlob("templates/*")
	adminGroup := gin.Group("/admin")
	adminGroup.Use(middleware.AuthenticationMiddleware(env.JWT.AccessTokenSecret))
	adminGroup.Use(middleware.RoleMiddleware(tokenutil.ROLE_ADMIN))
	// All admin APIs
	{
		dashboardRouter(userRepo, adminGroup)
	}
}

func swaggerRouter(group *gin.RouterGroup) {
	// Swagger API documentation
	url := ginSwagger.URL("./swagger/doc.json")
	group.GET("/swagger/*any",
		ginSwagger.WrapHandler(swaggerFiles.Handler, url),
	)
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
		metrics := userRepo.(*mongo.UserRepo).Metrics()

		if c.GetHeader(keyAccept) == contentTypeJSON {
			c.JSON(http.StatusOK, metrics)
			return
		}
		// Render HTML template with metrics data
		c.HTML(http.StatusOK, "dashboard.html", gin.H{
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
		c.HTML(http.StatusOK, "signup.html", nil)
	})

	group.POST("/signup", func(c *gin.Context) {
		var user domain.SignUpRequest

		// Check user input
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, &domain.APIResponse[any]{
				Status: http.StatusBadRequest, Message: "Invalid data, imposible to parse"})
			return
		}

		resp, _ := authController.Register(c, user)
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
		c.HTML(http.StatusOK, "login.html", nil)
	})

	group.POST("/login", func(c *gin.Context) {
		var user domain.LoginRequest
		accessToken := c.GetHeader(keyAuthorization)

		// Check user input
		if err := c.ShouldBindJSON(&user); err != nil && accessToken == "" {
			c.JSON(http.StatusBadRequest, &domain.APIResponse[any]{
				Status: http.StatusBadRequest, Message: "Invalid data, imposible to parse"})
			return
		}

		resp, _ := authController.Login(c, accessToken, user)
		if resp.Data != nil {
			c.Header(keyAuthorization, "Bearer "+resp.Data.AccessToken)
			// Set JWT in HttpOnly cookies (for web clients)
			c.SetCookie(cookieAccessToken, resp.Data.AccessToken,
				authController.GetAccessTokenExpirySeconds(), "/",
				"", false, true)
			c.SetCookie(cookieRefreshToken, resp.Data.RefreshToken,
				authController.GetRefreshTokenExpirySeconds(), "/",
				"", false, true)
		}
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
			role = tokenutil.ROLE_USER
		}

		resp, _ := authController.RefreshToken(c, refreshToken, role)
		if resp.Data != nil {
			c.Header(keyAuthorization, "Bearer "+resp.Data.AccessToken)
		}
		c.JSON(resp.Status, resp)
	})
}

// Profile handler returns teh logged user data
//
//	@Summary		Profile
//	@Description	Function for getting the user profile
//	@ID				profile
//	@Tags			Profile
//	@Security		Bearer JWT
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string				true	"Bearer JWT"	default(Bearer XXX)
//	@Success		200				{object}	map[string]string	"ok"
//	@Failure		400				{integer}	string				"not registered"
//	@Failure		404				{string}	integer				"not registered"
//	@Router			/protected/profile [get]
func profileRouter(authController *controller.AuthControl, group *gin.RouterGroup) {
	group.GET("/profile", func(c *gin.Context) {
		accessToken := c.GetHeader(keyAuthorization)
		middleware.ExtractCookieAccessToken(c, &accessToken)

		user, err := authController.GetUserByJWT(c, accessToken)

		if c.GetHeader(keyAccept) == contentTypeJSON {
			if err != nil {
				c.JSON(http.StatusNotFound, &domain.APIResponse[any]{
					Status: http.StatusNotFound, Message: "User not found"})
				return
			} else {
				c.JSON(http.StatusOK, user)
				return
			}
		}

		if err != nil {
			c.Redirect(http.StatusUnauthorized, "/login")
		}
		c.HTML(http.StatusOK, "profile.html", user)
	})
}

func NewTaskRouter(
	env *bootstrap.Env, timeout time.Duration, db domain.UserStore, group *gin.RouterGroup) {
	group.GET("/tasks", func(c *gin.Context) {
		// Task handler logic
	})
	group.POST("/tasks", func(c *gin.Context) {
		// Create task handler logic
	})
}
