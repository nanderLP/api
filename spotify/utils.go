package spotify

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int16  `json:"expires_in"`
	Scope        string `json:"scope"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type SavedCredentials struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func RequestAccessToken(authCode string) error {
	tokenUrl := url.URL{
		Scheme: "https",
		Host:   "accounts.spotify.com",
		Path:   "api/token",
	}
	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", authCode)
	params.Set("redirect_uri", os.Getenv("SPOTIFY_REDIRECT_URI"))
	paramsBytes := []byte(params.Encode())
	req, err := http.NewRequest("POST", tokenUrl.String(), bytes.NewBuffer(paramsBytes))

	if err != nil {
		return err
	}

	base64Auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v:%v", os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET"))))

	req.Header.Set("Authorization", "Basic "+base64Auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	tokenResponse := TokenResponse{}

	err = json.Unmarshal(body, &tokenResponse)

	if err != nil || tokenResponse.AccessToken == "" || tokenResponse.RefreshToken == "" {
		errorResponse := ErrorResponse{}
		json.Unmarshal(body, &errorResponse)
		errorMessage := "an error occured while parsing Spotify's OAuth response\n"
		if err != nil {
			errorMessage += err.Error() + "\n"
		}
		if errorResponse.Error != "" {
			errorMessage += errorResponse.Error
		}
		return errors.New(errorMessage)
	}

	savedCredentials := SavedCredentials{
		AccessToken:  tokenResponse.AccessToken,
		RefreshToken: tokenResponse.RefreshToken,
	}

	credentialsJson, err := json.Marshal(savedCredentials)

	fmt.Println(savedCredentials)

	fmt.Println(string(credentialsJson))

	if err != nil {
		return errors.New("could not parse credentials from Spotify's OAuth service")
	}

	err = os.WriteFile("./spotify_credentials.json", credentialsJson, 0644)

	if err != nil {
		return errors.New("could not save credentials")
	}

	return nil
}

func RefreshTokens(refreshToken string) error {
	// refresh credentials
	client := &http.Client{}

	tokenEndpoint := url.URL{
		Scheme: "https",
		Host:   "accounts.spotify.com",
		Path:   "api/token",
	}
	params := url.Values{}
	params.Set("grant_type", "refresh_token")
	params.Set("refresh_token", refreshToken)
	paramsBytes := []byte(params.Encode())
	req, err := http.NewRequest("POST", tokenEndpoint.String(), bytes.NewBuffer(paramsBytes))

	if err != nil {
		return errors.New("could not refresh session")
	}

	base64Auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v:%v", os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET"))))

	req.Header.Set("Authorization", "Basic "+base64Auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	tokenResponse := TokenResponse{}

	err = json.Unmarshal(body, &tokenResponse)

	if err != nil || tokenResponse.AccessToken == "" {
		errorResponse := ErrorResponse{}
		json.Unmarshal(body, &errorResponse)
		errorMessage := "An error occured while parsing Spotify's OAuth response\n"
		if err != nil {
			errorMessage += err.Error() + "\n"
		}
		if errorResponse.Error != "" {
			errorMessage += errorResponse.Error
		}
		return errors.New(errorMessage)
	}

	var newRefreshToken string
	if tokenResponse.RefreshToken != "" {
		newRefreshToken = tokenResponse.RefreshToken
	} else {
		newRefreshToken = refreshToken
	}

	savedCredentials := SavedCredentials{
		AccessToken:  tokenResponse.AccessToken,
		RefreshToken: newRefreshToken,
	}

	credentialsJson, err := json.Marshal(savedCredentials)

	if err != nil {
		return errors.New("could not parse credentials from Spotify's OAuth service")
	}

	err = os.WriteFile("./spotify_credentials.json", credentialsJson, 0644)

	if err != nil {
		return errors.New("could not save credentials")
	}

	return nil
}
