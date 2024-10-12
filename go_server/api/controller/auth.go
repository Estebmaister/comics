package controller

import (
	"net/http"
	"strings"

	"comics/domain"
	"comics/internal/tokenutil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	USER = "USER"
)

// @Summary			Login user
// @ModuleID  	signUp
// @Description	Login a user with basic credentials to receive an auth 'token' in the headers if successful
// @ID					user-login
// @Tags				User login
// @Accept			json
// @Produce			json
// @Param				Authorization	header		string						false	"Token"
// @Param				user					body			domain.UserLogin	true	"Login user"
// @Success			200						{string}	string		"ok"
// @Failure			400						{string}	string		"no ok"
// @Router			/public/login [post]
func Login(c *gin.Context) {
	var user domain.UserLogin

	// Check user credentials and generate a JWT
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data"})
		return
	}

	// TODO: Check if credentials are valid (replace this logic with real authentication)
	if user.Username == "user" && user.Password == "password" {
		// Generate a JWT
		token, err := tokenutil.GenerateToken(uuid.New())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": token})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
	}
}

// @Summary		Register new user
// @Description	Function for registering a new user (for demonstration purposes), receive a condirmation for success or failure
// @ID				user-register
// @Tags			User register
// @Accept		json
// @Produce		json
// @Param			username	query			string	true	"Username"
// @Param			password	query			string	true	"Password"
// @Success		201				{object}	map[string]string	"registered"
// @Failure		400 			{integer} string  	"not registered"
// @Failure		404 			{string} 	integer		"not registered"
// @Router			/public/register [post]
func Register(c *gin.Context) {
	var user domain.User
	var newID uuid.UUID

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data"})
		return
	}

	// TODO: securely hash passwords before storing them
	newID, err := uuid.NewV7FromReader(strings.NewReader(USER))
	if err != nil {
		newID = uuid.New() // Fallback to a random UUID if generation fails
	}
	user.ID = newID
	// Store the user in the database (replace this logic with real database operations)
	// ...

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}
