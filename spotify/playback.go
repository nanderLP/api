package spotify

import (
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

type PlaybackResponse struct {
	Timestamp int64  `json:"timestamp"`
	Progress  int  `json:"progress_ms"`
	Playing   bool `json:"is_playing"`
	Item any `json:"item"`
}

func Playback(c *fiber.Ctx) error {
	// load credentials
	credentialsFile, err := os.ReadFile("./spotify_credentials.json")

	if err != nil {
		return c.Status(500).SendString("Could not load credentials")
	}

	credentials := SavedCredentials{}

	err = json.Unmarshal(credentialsFile, &credentials)

	if err != nil {
		return c.Status(500).SendString("Could not parse credentials")
	}

	if credentials.AccessToken == "" || credentials.RefreshToken == "" {
		return c.Status(500).SendString("Missing credentials")
	}

	endpoint := url.URL{
		Scheme: "https",
		Host:   "api.spotify.com",
		Path:   "v1/me/player/currently-playing",
	}

	req, err := http.NewRequest("GET", endpoint.String(), nil)

	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	req.Header.Add("Authorization", "Bearer "+credentials.AccessToken)

	client := &http.Client{}

	resp, err := client.Do(req)

	if resp.StatusCode == http.StatusUnauthorized {
		err = RefreshTokens(credentials.RefreshToken)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		// retry request
		resp, err = client.Do(req)
	}

	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return c.Status(500).SendString("Failed request, Code: " + string(resp.StatusCode))
	}

	apiResponse := PlaybackResponse{}

	err = json.Unmarshal(body, &apiResponse)

	if err != nil {
		return c.Status(500).SendString("Could not build response object")
	}

	return c.JSON(apiResponse)
}
