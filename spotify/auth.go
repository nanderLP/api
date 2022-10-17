package spotify

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"net/url"
	"os"
	"strings"
)

func Auth(c *fiber.Ctx) error {
	authError := c.Query("error")

	// callback error
	if authError != "" {
		return c.JSON(authError)
	}

	authCode := c.Query("code")

	// successful callback
	if authCode != "" {
		err := RequestAccessToken(authCode)

		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		return c.Status(201).SendString("Saved new credentials")
	} else {
		// build auth link and redirect
		scope := []string{"user-read-playback-state"}
		scopeString := strings.Join(scope[:], " ")
		params := fmt.Sprintf("response_type=%v&client_id=%v&redirect_uri=%v&scope=%v", "code", os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_REDIRECT_URI"), scopeString)
		authUrl := url.URL{
			Scheme:   "https",
			Host:     "accounts.spotify.com",
			Path:     "authorize",
			RawQuery: params,
		}
		return c.Redirect(authUrl.String())
	}
}
