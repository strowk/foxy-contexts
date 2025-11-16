package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/strowk/foxy-contexts/pkg/app"
	"github.com/strowk/foxy-contexts/pkg/auth"
	"github.com/strowk/foxy-contexts/pkg/fxctx"
	"github.com/strowk/foxy-contexts/pkg/mcp"
	"github.com/strowk/foxy-contexts/pkg/session"
	"github.com/strowk/foxy-contexts/pkg/streamable_http"

	"golang.org/x/oauth2"
)

type MySessionData struct {
	isItGreat bool
}

func (m *MySessionData) String() string {
	return "MySessionData"
}

// This example defines my-great-tool tool for MCP server that is using streamable http transport with authentication via OAuth2.

// --8<-- [start:tool]
func NewGreatTool(sm *session.SessionManager) fxctx.Tool {
	return fxctx.NewTool(
		// This information about the tool would be used when it is listed:
		&mcp.Tool{
			Name:        "my-great-tool",
			Description: Ptr("The great tool"),
			InputSchema: mcp.ToolInputSchema{ // here we tell client what we expect as input
				Type:       "object",
				Properties: map[string]map[string]interface{}{},
				Required:   []string{},
			},
		},

		// This is the callback that would be executed when the tool is called:
		func(ctx context.Context, args map[string]interface{}) *mcp.CallToolResult {
			data := sm.GetSessionData(ctx)
			if data == nil {
				sm.SetSessionData(ctx, &MySessionData{
					isItGreat: true,
				})
			}

			resp := "saving greatness to session"
			if data != nil {
				resp = "already great"
			}
			// here we can do anything we want
			return &mcp.CallToolResult{
				Content: []interface{}{
					mcp.TextContent{
						Type: "text",
						Text: fmt.Sprintf("Sup, %s", resp),
					},
				},
			}
		},
	)
}

// --8<-- [end:tool]

// --8<-- [start:server]
func main() {
	secret := "ZXhhbXBsZS1hcHAtc2VjcmV0"
	redirectBackHere := "http://localhost:8080/callback"
	conf := &oauth2.Config{
		ClientID:     "example-app",
		ClientSecret: secret,
		RedirectURL:  redirectBackHere,
		Scopes:       []string{"openid"},
		Endpoint: oauth2.Endpoint{
			AuthURL:   "http://localhost:5556/dex/auth",
			TokenURL:  "http://localhost:5556/dex/token",
			AuthStyle: oauth2.AuthStyleInHeader,
		},
	}

	// note: this would make this non-scalable, as we are storing
	// the state in memory, but this is just an example.
	// To make this production grade, we should store the state in a
	// shared storage, probably with some TTL as well
	nonces := make(map[string]string)
	// tokens := make(map[*oauth2Client.Token]struct{})

	server := app.
		NewBuilder().
		// adding the tool to the app
		WithTool(NewGreatTool).
		// setting up server
		WithName("great-tool-server").
		WithVersion("0.0.1").
		WithTransport(
			streamable_http.NewTransport(
				streamable_http.Endpoint{
					Hostname: "localhost",
					Port:     8080,
					Path:     "/mcp",

					AuthHost:   "localhost:8080",
					AuthScheme: "http",
				},
				streamable_http.EchoConfigurer{
					Configure: func(e *echo.Echo) {
						e.GET("/callback", func(c echo.Context) error {
							r := c.Request()
							code := r.URL.Query().Get("code")

							if code != "" {
								state := r.URL.Query().Get("state")

								if state == "" {
									log.Printf("state is empty")
									return errors.New("state is empty")
								}

								decodedState := make(map[string]string)
								err := json.Unmarshal([]byte(state), &decodedState)
								if err != nil {
									return err
								}
								nonce := decodedState["nonce"]
								authUrl := decodedState["auth_url"]
								if authUrl == "" {
									log.Printf("auth_url is empty")
									return errors.New("auth_url is empty")
								}
								if nonce == "" {
									log.Printf("nonce is empty")
									return errors.New("nonce is empty")
								}
								if nonces[nonce] != authUrl {
									log.Printf("nonce is not valid")
									return errors.New("nonce is not valid")
								}
								delete(nonces, nonce)

								tok, err := conf.Exchange(context.Background(), code)
								if err != nil {
									return err
								}

								log.Println("token: ", tok.AccessToken)

								// tokens[tok] = struct{}{}

								// redirect back to finish the flow
								return c.Redirect(http.StatusFound, "http://localhost:8080"+authUrl+"&remote_token="+tok.AccessToken)
							} else {
								log.Println("code is empty")
								c.Response().Writer.WriteHeader(http.StatusBadRequest)
							}
							return nil
						})
					},
				},
			),
		).
		WithAuthorization(auth.Must(auth.NewOauth2Authorization(
			auth.WithExcludedPaths("/callback"),
			auth.WithAuthorizationHandler(
				func(w http.ResponseWriter, r *http.Request) (userID string, err error) {

					// TODO: does it actually make sense to save remote token as user id?
					remoteToken := r.URL.Query().Get("remote_token")
					if remoteToken != "" {
						return remoteToken, nil
					}

					authUrl := r.URL.String()
					nonce := uuid.NewString()
					nonces[nonce] = authUrl

					state := struct {
						AuthUrl string `json:"auth_url"`
						Nonce   string `json:"nonce"`
					}{
						AuthUrl: authUrl,
						Nonce:   nonce,
					}

					marshalledState, err := json.Marshal(state)
					if err != nil {
						log.Println("Error marshalling state: ", err)
						return "", err
					}

					url := conf.AuthCodeURL(string(marshalledState))

					http.Redirect(w, r, url, http.StatusFound)
					return "", nil // this will stop the request here as we already provided redirection
				},
			),
		)))

	err := server.Run()
	if err != nil {
		if err == http.ErrServerClosed {
			log.Println("Server closed")
		} else {
			log.Fatalf("Server error: %v", err)
		}
	}
}

// --8<-- [end:server]

func Ptr[T any](v T) *T {
	return &v
}
