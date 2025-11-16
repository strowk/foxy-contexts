package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/grokify/go-pkce"
	"github.com/labstack/echo/v4"
)

func main() {
	redirectUri := "http://localhost:8081/callback"

	registerClientBody := `{
		"client_name": "test-client-name",
		"redirect_uris": ["%s"],
		"token_endpoint_auth_method": "client_secret_post",
		"grant_types": ["authorization_code"],
		"response_types": ["code"]
	}`
	registerClientBody = fmt.Sprintf(registerClientBody, redirectUri)

	reqBody := []byte(registerClientBody)

	req, err := http.NewRequest("POST", "http://localhost:8080/register", bytes.NewBuffer(reqBody))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	respBody := new(bytes.Buffer)
	_, err = respBody.ReadFrom(resp.Body)
	if err != nil {
		panic(err)
	}
	bodyString := respBody.String()

	if resp.StatusCode != http.StatusOK {
		panic("Error: " + http.StatusText(resp.StatusCode) + " " + bodyString)
	} else {
		println("Response: " + bodyString)
	}

	var registeredClient struct {
		ClientId     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}

	err = json.Unmarshal(respBody.Bytes(), &registeredClient)
	if err != nil {
		panic(err)
	}

	codeVerifier, err := pkce.NewCodeVerifier(-1)
	if err != nil {
		panic(err)
	}
	codeChallenge := pkce.CodeChallengeS256(codeVerifier)

	resultingUrl := fmt.Sprintf(`http://localhost:8080/authorize?client_id=%s&response_type=code&code_challenge=%s&redirect_uri=%s&code_challenge_method=S256`, registeredClient.ClientId, codeChallenge, redirectUri)
	println("Go to URL: " + resultingUrl)

	e := echo.New()
	e.GET("/callback", func(c echo.Context) error {
		code := c.QueryParam("code")
		if code == "" {
			return c.String(http.StatusBadRequest, "code is empty")
		}
		resp, err := http.PostForm("http://localhost:8080/token", url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"redirect_uri":  {redirectUri},
			"code_verifier": {codeVerifier},
			"client_id":     {registeredClient.ClientId},
			"client_secret": {registeredClient.ClientSecret},
		})
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error sending request: "+err.Error())
		}

		defer resp.Body.Close()
		respBody := new(bytes.Buffer)
		_, err = respBody.ReadFrom(resp.Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error reading response: "+err.Error())
		}
		bodyString := respBody.String()
		if resp.StatusCode != http.StatusOK {
			return c.String(http.StatusInternalServerError, "Error: "+http.StatusText(resp.StatusCode)+" "+bodyString)
		} else {
			println("Token Response: " + bodyString)

			parsedToken := new(struct {
				AccessToken  string `json:"access_token"`
				TokenType    string `json:"token_type"`
				ExpiresIn    int    `json:"expires_in"`
				RefreshToken string `json:"refresh_token"`
			})

			err = json.Unmarshal(respBody.Bytes(), parsedToken)
			if err != nil {
				return c.String(http.StatusInternalServerError, "Error unmarshalling response: "+err.Error())
			}

			// requesting MCP ping with the access token
			bodyString, statusCode, err := callPing(parsedToken.AccessToken, client)
			if err != nil {
				return c.String(http.StatusInternalServerError, "Error calling MCP ping: "+err.Error())
			}
			if statusCode != http.StatusOK {
				return c.String(http.StatusInternalServerError, fmt.Sprintf("error: %s %s", http.StatusText(statusCode), bodyString))
			} else {
				println("MCP Ping Response: " + bodyString)
			}

			// try same request without authorization header
			// to check that authorization is required
			bodyString, statusCode, err = callPing("", client)
			if err != nil {
				return c.String(http.StatusInternalServerError, "Error calling MCP ping: "+err.Error())
			}

			if statusCode != http.StatusUnauthorized {
				println("Expected 401 Unauthorized, got: " + http.StatusText(statusCode))
			} else {
				println("MCP Ping Response without auth: " + http.StatusText(statusCode))
			}
		}
		return c.Redirect(http.StatusFound, "/ok")
	})

	e.GET("/ok", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	e.Start(":8081")
}

func callPing(token string, client *http.Client) (string, int, error) {
	req, err := http.NewRequest("POST", "http://localhost:8080/mcp", bytes.NewBuffer([]byte(`{"method":"ping","params":{},"id":0, "jsonrpc":"2.0"}`)))
	if err != nil {
		return "", 0, fmt.Errorf("creating request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()
	respBody := new(bytes.Buffer)
	_, err = respBody.ReadFrom(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("reading response: %w", err)
	}
	bodyString := respBody.String()

	return bodyString, resp.StatusCode, nil
}
