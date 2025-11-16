package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/manage"
	"github.com/go-oauth2/oauth2/v4/models"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/go-oauth2/oauth2/v4/store"
)

var (
	ErrInvalidRedirectURI = errors.New("invalid redirect URI")
)

// Authorization is an interface that defines necessary methods for setting up
// an authorization mechanism for MCP server.
//
// Authorization would only work if the server is configured to use a transport
// that supports authorization, which at the moment is only Streamable HTTP, see
// more here: https://spec.modelcontextprotocol.io/specification/2025-03-26/basic/authorization/
type Authorization interface {
	// IsRequired returns true if the authorization is required for the server.
	// When true, the server will respond to all requests without authorization with
	// an unauthorized error (401 HTTP status code).
	IsRequired() bool

	RegisterClient(ctx context.Context, req *ClientRegistrationRequest) (*ClientRegistrationResponse, error)

	HandleAuthorizeRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) error
	HandleTokenRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) error

	ValidateToken(ctx context.Context, token string) (string, error)

	ExcludedPaths() []string
}

type AuthorizationHandler func(w http.ResponseWriter, r *http.Request) (userID string, err error)

type oauth2Auth struct {
	required bool

	registeredClients registeredClients

	manager     *manage.Manager
	clientStore *store.ClientStore
	server      *server.Server

	authorizationHandler AuthorizationHandler

	excludedPaths []string
}

func Must(auth Authorization, err error) Authorization {
	if err != nil {
		panic(err)
	}
	return auth
}

// NewOauth2Authorization creates a new Oauth2Authorization instance with the given options.
//
// See list of options in ./options.go file.
func NewOauth2Authorization(opts ...Oauth2AuthorizationOption) (*oauth2Auth, error) {
	auth := &oauth2Auth{
		required: true,
		registeredClients: registeredClients{
			Clients: sync.Map{},
		},
		manager: manage.NewDefaultManager(),
	}

	// this thing would be potentially broken if two clients have put first redirect uri as the same
	// theoretically could support multiple redirect uris for the same client, but only
	// if go-auth2 would give us client id for this function, but it does not ATM
	//
	// auth.manager.SetValidateURIHandler(func(baseRedirectURI, redirectURI string) error {
	// 	for _, client := range auth.registeredClients.Clients {
	// 		if client.redirectUris[0] == baseRedirectURI {
	// 			for _, uri := range client.redirectUris {
	// 				if uri == redirectURI {
	// 					return nil
	// 				}
	// 			}
	// 		}
	// 	}
	// 	return ErrInvalidRedirectURI
	// })

	tokenStore, err := store.NewMemoryTokenStore()
	if err != nil {
		return nil, err
	}
	auth.manager.MapTokenStorage(tokenStore)

	auth.clientStore = store.NewClientStore()
	auth.manager.MapClientStorage(auth.clientStore)

	serverCfg := server.NewConfig()
	serverCfg.ForcePKCE = true
	serverCfg.AllowedCodeChallengeMethods = []oauth2.CodeChallengeMethod{
		oauth2.CodeChallengeS256,
	}

	auth.server = server.NewServer(serverCfg, auth.manager)
	auth.server.SetAllowGetAccessRequest(true)
	auth.server.SetClientInfoHandler(server.ClientFormHandler)
	auth.server.SetAllowedGrantType("authorization_code")
	auth.server.SetAllowedResponseType("code")

	// auth.manager.MapAccessGenerate()

	for _, opt := range opts {
		opt(auth)
	}

	if auth.authorizationHandler != nil {
		auth.server.SetUserAuthorizationHandler(func(w http.ResponseWriter, r *http.Request) (string, error) {
			// log.Println("Authorization handler called")
			// TODO: issue session and return it as a user id
			return auth.authorizationHandler(w, r)
		})
	}

	return auth, nil
}

func (o *oauth2Auth) IsRequired() bool {
	return o.required
}

type Oauth2AuthorizationOption func(*oauth2Auth)

type AuthorizeableTransport interface {
	PlugInAuthorization(authorization Authorization) error
}

func (o *oauth2Auth) RegisterClient(ctx context.Context, req *ClientRegistrationRequest) (*ClientRegistrationResponse, error) {
	client, err := o.registeredClients.registerClient(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to register client: %w", err)
	}

	err = o.clientStore.Set(client.clientId, &models.Client{
		ID:     client.clientId,
		Secret: client.clientSecret,
		Domain: client.redirectUris[0],
	})
	if err != nil {
		return nil, fmt.Errorf("failed to store client: %w", err)
	}

	return &ClientRegistrationResponse{
		ClientId:     client.clientId,
		ClientSecret: client.clientSecret,
	}, nil
}

func (o *oauth2Auth) HandleAuthorizeRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	log.Println("HandleAuthorizeRequest called")
	return o.server.HandleAuthorizeRequest(w, r)
}

func (o *oauth2Auth) HandleTokenRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	return o.server.HandleTokenRequest(w, r)
}

func (o *oauth2Auth) ValidateToken(ctx context.Context, token string) (string, error) {
	tk, err := o.manager.LoadAccessToken(ctx, token)
	if err != nil {
		return "", err
	}
	return tk.GetUserID(), nil
}

func (o *oauth2Auth) ExcludedPaths() []string {
	return o.excludedPaths
}
