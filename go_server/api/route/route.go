package route

import (
	"net/http"
	"time"

	"comics/api/controller"
	"comics/api/middleware"
	"comics/bootstrap"
	"comics/domain"
	"comics/mongo"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Setup(env *bootstrap.Env, timeout time.Duration, db mongo.Database, gin *gin.Engine) {

	// All Public APIs
	publicRouter := gin.Group("/")
	{
		swaggerRouter(env, timeout, db, publicRouter)
		signUpRouter(env, timeout, db, publicRouter)
		loginRouter(env, timeout, db, publicRouter)
		NewRefreshTokenRouter(env, timeout, db, publicRouter)
	}

	protectedRouter := gin.Group("/protected")
	// Middleware to verify AccessToken
	protectedRouter.Use(middleware.AuthenticationMiddleware(env.AccessTokenSecret))
	// All Private APIs
	NewProfileRouter(env, timeout, db, protectedRouter)
	NewTaskRouter(env, timeout, db, protectedRouter)

	adminGroup := gin.Group("/admin")
	adminGroup.Use(middleware.AuthenticationMiddleware(env.AccessTokenSecret))
	adminGroup.Use(middleware.RoleMiddleware("admin"))
	dashboardRouter(env, timeout, db, adminGroup)
}

// @Summary		Dashboard
// @Description	Function for getting the admin dashboard
// @ID				dashboard
// @Tags			Dashboard
// @Accept		json
// @Produce		json
// @Param			Authorization	header		string		true	"Bearer <JWT Token>"
// @Success		200				{object}	map[string]string	"ok"
// @Failure		400 			{integer} string  	"not registered"
// @Failure		404 			{string} 	integer		"not registered"
// @Router		/admin/dashboard [get]
func dashboardRouter(_ *bootstrap.Env, _ time.Duration, _ mongo.Database, group *gin.RouterGroup) {
	group.GET("/dashboard", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Dashboard"})
	})
}

func swaggerRouter(env *bootstrap.Env, _ time.Duration, _ mongo.Database, group *gin.RouterGroup) {
	// Swagger API documentation
	url := ginSwagger.URL("./swagger/doc.json")
	println("http://" + env.ServerAddress + "/swagger/index.html")
	group.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))
}

// @Summary		SignUp new user
// @Description	Function for SigningUp a new user (for demonstration purposes), receive a confirmation for success or failure
// @ID				user-signup
// @Tags			Authentication
// @Accept		json
// @Produce		json
// @Param			username	query			string			false	"Username"	default(testuser)
// @Param			user			body			domain.User	true	"Login user"
// @Success		201				{object}	map[string]string	"registered"
// @Failure		400 			{integer} string  	"not registered"
// @Failure		404 			{string} 	integer		"not registered"
// @Router		/signup [post]
func signUpRouter(_ *bootstrap.Env, _ time.Duration, _ mongo.Database, group *gin.RouterGroup) {
	group.POST("/signup", func(c *gin.Context) {
		var user domain.User

		// Check user input
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data, imposible to parse"})
			return
		}

		user.Role = "user"
		msg, status, err := controller.Register(user)
		if err != nil {
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.JSON(status, msg)
	})
}

// @Summary			Login existent user
// @ModuleID  	signUp
// @Description	Login a user with basic credentials to receive an auth 'token' in the headers if successful
// @ID					user-login
// @Tags				Authentication
// @Accept			json
// @Produce			json
// @Param				Authorization	header		string						  false	"Bearer <JWT Token>"
// @Param				user					body			domain.LoginRequest	true	"Login user"
// @Success			200						{string}	string		"ok"
// @Failure			400						{string}	string		"no ok"
// @Router			/login [post]
func loginRouter(env *bootstrap.Env, _ time.Duration, _ mongo.Database, group *gin.RouterGroup) {
	group.POST("/login", func(c *gin.Context) {
		var user domain.LoginRequest

		// Check user input
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data, imposible to parse"})
			return
		}

		msg, status, err := controller.Login(c.GetHeader("Authorization"), env, user)
		if err != nil {
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		c.Header("Authorization", "Bearer "+msg["access_token"].(string))
		c.JSON(status, msg)
	})
}

// @Summary		RefreshToken
// @Description	Function for refreshing the access token
// @ID				refresh-token
// @Tags			Authentication
// @Accept		json
// @Produce		json
// @Param			Authorization	header		string		true	"Bearer <JWT Token>"
// @Success		200				{object}	map[string]string	"ok"
// @Failure		400 			{integer} string  	"not registered"
// @Failure		404 			{string} 	integer		"not registered"
// @Router		/refresh-token [post]
func NewRefreshTokenRouter(env *bootstrap.Env, timeout time.Duration, db mongo.Database, group *gin.RouterGroup) {
	group.POST("/refresh-token", func(c *gin.Context) {
		// Refresh token handler logic
	})
}

// @Summary		Profile
// @Description	Function for getting the user profile
// @ID				profile
// @Tags			Profile
// @Accept		json
// @Produce		json
// @Param			Authorization	header		string		true	"Bearer <JWT Token>"
// @Success		200				{object}	map[string]string	"ok"
// @Failure		400 			{integer} string  	"not registered"
// @Failure		404 			{string} 	integer		"not registered"
// @Router		/protected/profile [get]
func NewProfileRouter(env *bootstrap.Env, timeout time.Duration, db mongo.Database, group *gin.RouterGroup) {
	group.GET("/profile", func(c *gin.Context) {
		// Profile handler logic
	})
}

func NewTaskRouter(env *bootstrap.Env, timeout time.Duration, db mongo.Database, group *gin.RouterGroup) {
	group.GET("/tasks", func(c *gin.Context) {
		// Task handler logic
	})
	group.POST("/tasks", func(c *gin.Context) {
		// Create task handler logic
	})
}
