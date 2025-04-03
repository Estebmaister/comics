package route

import (
	"net/http"

	"comics/api/controller"
	"comics/api/middleware"
	"comics/domain"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// GetProfile handler returns the logged user data
//
//	@Summary		GetProfile
//	@Description	Endpoint for getting the logged user profile
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
func getProfile(ac *controller.AuthControl) func(c *gin.Context) {
	return func(c *gin.Context) {
		accessToken := c.GetHeader(keyAuthorization)
		middleware.ExtractCookieAccessToken(c, &accessToken)

		user, err := ac.GetUserByJWT(c.Request.Context(), accessToken)

		// If call comes from api return JSON
		if c.GetHeader(keyAccept) == contentTypeJSON {
			if err != nil {
				c.Error(err) // nolint:errcheck
				c.JSON(http.StatusNotFound, &domain.APIResponse[any]{
					Status: http.StatusNotFound, Message: "User not found"})
				return
			}

			c.JSON(http.StatusOK, user)
			return
		}

		// If call comes from browser render profile or redirect to login if no user found
		if err != nil {
			c.Error(err) // nolint:errcheck
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
		otelgin.HTML(c, http.StatusOK, "profile.html", user)
	}
}

// UpdateProfile updates a user's profile
//
//	@Summary		UpdateProfile
//	@Description	Endpoint for updating the logged user profile
//	@ID				update-profile
//	@Tags			Profile
//	@Security		Bearer JWT
//	@Accept			json
//	@Produce		json
//	@Param			Authorization	header		string					true	"Bearer JWT"	default(Bearer XXX)
//	@Param			user			body		domain.SignUpRequest	true	"Update user"
//	@Success		200				{object}	map[string]string		"ok"
//	@Failure		400				{integer}	string					"not registered"
//	@Failure		404				{string}	integer					"not registered"
//	@Router			/protected/profile [put]
func putProfile(ac *controller.AuthControl) func(c *gin.Context) {
	return func(c *gin.Context) {
		accessToken := c.GetHeader(keyAuthorization)
		middleware.ExtractCookieAccessToken(c, &accessToken)

		var user domain.UpdateRequest
		if err := c.ShouldBindJSON(&user); err != nil {
			c.Error(err) // nolint:errcheck
			c.JSON(http.StatusBadRequest, &domain.NoDataResponse{
				Status: http.StatusBadRequest, Message: "Invalid data, imposible to parse"})
			return
		}

		// Validate at least one field is provided
		if err := user.Validate(); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := ac.UpdateProfile(c.Request.Context(), accessToken, user)
		if err != nil {
			c.Error(err) // nolint:errcheck
			c.JSON(resp.Status, resp)
			return
		}
		c.JSON(resp.Status, resp)
	}
}
