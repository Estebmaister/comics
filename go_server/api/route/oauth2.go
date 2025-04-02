package route

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"comics/api/controller"
	"comics/bootstrap"
	"comics/domain"
	"comics/internal/repo"
	"comics/sampler"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	oauthStateString = sampler.RandomString() // A random string for CSRF protection
	googleTokenURL   = "https://www.googleapis.com/oauth2/v2/userinfo?access_token="
	logInPath        = "/login"
	signUpPath       = "/signup"
	callbackPath     = "/auth/callback"
)

var googleOauthConfig = &oauth2.Config{
	Scopes: []string{
		"https://www.googleapis.com/auth/userinfo.profile",
		"https://www.googleapis.com/auth/userinfo.email",
	},
	Endpoint: google.Endpoint,
}

// setOAuth2 configures the OAuth2 routes
func setOAuth2(env *bootstrap.Env, ac *controller.AuthControl, g *gin.RouterGroup) {

	// OAuth2 Google configuration
	googleOauthConfig.ClientID = env.GoogleClientID
	googleOauthConfig.ClientSecret = env.GoogleClientSecret

	redirectURL := env.HostURL + callbackPath
	if strings.Contains(env.HostURL, "localhost") || strings.Contains(env.HostURL, "0.0.0.0") {
		redirectURL = "http://" + redirectURL
	} else {
		redirectURL = "https://" + redirectURL
	}

	googleOauthConfig.RedirectURL = redirectURL

	g.GET("/auth/google", func(c *gin.Context) {
		url := googleOauthConfig.AuthCodeURL(oauthStateString)
		c.Redirect(http.StatusTemporaryRedirect, url)
	})

	g.GET(callbackPath, handleGoogleCallback(ac))
}

// handleGoogleCallback handles the callback from Google OAuth2
func handleGoogleCallback(ac *controller.AuthControl) func(c *gin.Context) {
	return func(c *gin.Context) {

		googleDTO := extractGoogleDTO(c)
		if googleDTO == nil {
			c.Redirect(http.StatusTemporaryRedirect, logInPath)
			return
		}

		resp, err := ac.LoginByOAuthEmail(c.Request.Context(), googleDTO.Email)
		if errors.Is(err, repo.ErrNotFound) {
			resp, err = ac.Register(c.Request.Context(), domain.SignUpRequest{
				Email: googleDTO.Email,
			})
		}
		if err != nil {
			log.Err(err).Msg("Failed logging in by email")
			c.Redirect(http.StatusTemporaryRedirect, signUpPath)
			return
		}

		// Set JWT in HttpOnly cookies
		c.SetCookie(cookieAccessToken, resp.Data.AccessToken,
			ac.GetAccessTokenExpirySeconds(), "/",
			"", false, true)
		c.SetCookie(cookieRefreshToken, resp.Data.RefreshToken,
			ac.GetRefreshTokenExpirySeconds(), "/",
			"", false, true)
		c.Redirect(http.StatusTemporaryRedirect, "/protected/profile")
	}
}

type googleDTO struct {
	Email         string `json:"email"`
	FamilyName    string `json:"family_name"`
	GivenName     string `json:"given_name"`
	Id            string `json:"id"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
}

// extractGoogleDTO extracts the Google user information from the request
func extractGoogleDTO(c *gin.Context) *googleDTO {
	state := c.Request.FormValue("state")
	if state != oauthStateString { // CSRF protection validation
		log.Error().Msgf("Invalid oauth state: %s", state)
		return nil
	}

	code := c.Request.FormValue("code")
	token, err := googleOauthConfig.Exchange(c.Request.Context(), code)
	if err != nil {
		log.Err(err).Msg("Failed code exchange")
		return nil
	}

	response, err := http.Get(googleTokenURL + token.AccessToken)
	if err != nil {
		log.Err(err).Msgf("Failed getting user info, token: %v", token)
		return nil
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		log.Err(err).Msg("Failed reading user info")
		return nil
	}

	var googleDTO googleDTO
	if err := json.Unmarshal(data, &googleDTO); err != nil {
		log.Err(err).Msg("Failed unmarshalling user info")
		return nil
	}
	log.Debug().RawJSON("body", data).Msg("OAuth2 User info")
	return &googleDTO
}
