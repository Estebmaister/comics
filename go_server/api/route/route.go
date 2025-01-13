package route

import (
	"net/http"
	"time"

	"comics/api/controller"
	"comics/api/middleware"
	"comics/bootstrap"
	"comics/docs"
	"comics/domain"
	"comics/internal/tokenutil"
	"comics/mongo"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var ( // Headers
	Authorization = "Authorization"
	Role          = "Role"
)

func Setup(
	env *bootstrap.Env, timeout time.Duration, db mongo.Database, gin *gin.Engine) {
	basePath := "/"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}
	docs.SwaggerInfo.Host = env.ServerAddress
	docs.SwaggerInfo.BasePath = basePath
	publicRouter := gin.Group(basePath)
	// All Public APIs
	{
		swaggerRouter(env, timeout, db, publicRouter)
		signUpRouter(env, timeout, db, publicRouter)
		loginRouter(env, timeout, db, publicRouter)
		refreshTokenRouter(env, timeout, db, publicRouter)
	}

	protectedRouter := gin.Group("/protected")
	// Middleware to verify AccessToken
	protectedRouter.Use(middleware.AuthenticationMiddleware(env.AccessTokenSecret))
	// All protected Private APIs
	{
		NewProfileRouter(env, timeout, db, protectedRouter)
		NewTaskRouter(env, timeout, db, protectedRouter)
	}

	adminGroup := gin.Group("/admin")
	adminGroup.Use(middleware.AuthenticationMiddleware(env.AccessTokenSecret))
	adminGroup.Use(middleware.RoleMiddleware(tokenutil.ROLE_ADMIN))
	// All admin APIs
	{
		dashboardRouter(env, timeout, db, adminGroup)
	}
}

func swaggerRouter(
	env *bootstrap.Env, _ time.Duration, _ mongo.Database, group *gin.RouterGroup) {
	// Swagger API documentation
	url := ginSwagger.URL("./swagger/doc.json")
	println("http://" + env.ServerAddress + "/swagger/index.html")
	group.GET(
		"/swagger/*any",
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
func dashboardRouter(
	_ *bootstrap.Env, _ time.Duration, _ mongo.Database, group *gin.RouterGroup) {
	group.GET("/dashboard", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Dashboard"})
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
//	@Failure		400		{string}	string					"not registered, invalid data"
//	@Failure		409		{string}	string					"username or email already in use"
//	@Router			/signup [post]
func signUpRouter(
	_ *bootstrap.Env, _ time.Duration, _ mongo.Database, group *gin.RouterGroup) {
	group.POST("/signup", func(c *gin.Context) {
		var user domain.SignUpRequest

		// Check user input
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(
				http.StatusBadRequest, gin.H{"error": "Invalid data, imposible to parse"})
			return
		}

		resp, status, err := controller.Register(c, user)
		if err != nil {
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(status, resp)
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
func loginRouter(
	env *bootstrap.Env, _ time.Duration, _ mongo.Database, group *gin.RouterGroup) {
	group.POST("/login", func(c *gin.Context) {
		var user domain.LoginRequest
		accessToken := c.GetHeader(Authorization)

		// Check user input
		if err := c.ShouldBindJSON(&user); err != nil && accessToken == "" {
			c.JSON(
				http.StatusBadRequest, gin.H{"error": "Invalid data, imposible to parse"})
			return
		}

		resp, status, err := controller.Login(c, env, accessToken, user)
		if err != nil {
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.Header(Authorization, "Bearer "+resp.AccessToken)
		c.JSON(status, resp)
	})
}

// refreshTokenRouter tries to renovate the credentials of a logged user
//
//	@Summary		RefreshToken
//	@Description	Function for refreshing the access token
//	@ID				refresh-token
//	@Tags			Authentication
//	@Security		Bearer JWT
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string				true	"Bearer JWT"	default(Bearer XXX)
//	@Param			Role			header		string				false	"role"			Enums(user, admin)
//	@Success		200				{object}	domain.AuthResponse	"new access token generated"
//	@Header			200				{string}	Authorization		"Bearer JWT"
//	@Failure		400				{integer}	string				"not registered"
//	@Failure		404				{string}	integer				"not registered"
//	@Router			/refresh-token [post]
func refreshTokenRouter(
	env *bootstrap.Env, _ time.Duration, _ mongo.Database, group *gin.RouterGroup) {
	group.POST("/refresh-token", func(c *gin.Context) {
		role := c.GetHeader(Role)
		if role == "" {
			role = tokenutil.ROLE_USER
		}

		resp, status, err := controller.RefreshToken(
			c, env, c.GetHeader(Authorization), role)
		if err != nil {
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.Header(Authorization, "Bearer "+resp.AccessToken)
		c.JSON(status, resp)
	})
}

// Profile handler
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
func NewProfileRouter(
	env *bootstrap.Env, timeout time.Duration, db mongo.Database, group *gin.RouterGroup) {
	group.GET("/profile", func(c *gin.Context) {
		// Profile handler logic
	})
}

func NewTaskRouter(
	env *bootstrap.Env, timeout time.Duration, db mongo.Database, group *gin.RouterGroup) {
	group.GET("/tasks", func(c *gin.Context) {
		// Task handler logic
	})
	group.POST("/tasks", func(c *gin.Context) {
		// Create task handler logic
	})
}
