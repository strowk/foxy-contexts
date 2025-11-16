package auth

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"sync"

	"github.com/google/uuid"
)

var (
	ErrInvalidRedirectUri     = errors.New("invalid redirect URI - must be absolute and either use https scheme or be local")
	ErrInvalidTokenAuthMethod = errors.New("invalid token endpoint auth method - only client_secret_post is supported")
	ErrInvalidGrantType       = errors.New("invalid grant type - only authorization_code is supported")
	ErrInvalidResponseType    = errors.New("invalid response type - only code is supported")

	ErrInvalidJWKs = errors.New("only one of jwks_uri or jwks must be provided")

	GrantTypesSupported = []string{"authorization_code"} // TODO: support refresh token and maybe client creds as well

	ResponseTypesSupported                     = []string{"code"}
	ResponseModesSupported                     = []string{"form_post"}
	TokenEndpointAuthMethodsSupported          = []string{"client_secret_post"}
	TokenEndpointAuthSigningAlgValuesSupported = []string{"RS256"}
	CodeChallengeMethodsSupported              = []string{"S256"}
)

type oauthClient struct {
	clientId     string
	clientSecret string
	redirectUris []string
}

// TODO: should also support persisent storage of clients, so that they can be reused across server restarts?
// TODO: need to consider if registration request with same redirect uri should be mapped to the same client id? (probably only if not loopback..)

type registeredClients struct {
	Clients sync.Map
}

type ClientRegistrationRequest struct {
	ClientName              *string  `json:"client_name"`
	RedirectUris            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod *string  `json:"token_endpoint_auth_method"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	// TODO: add other remaining fields from RFC 7591
}

func (req *ClientRegistrationRequest) validate() error {
	err := req.validateTokenEndpointAuthMethod()
	if err != nil {
		return err
	}
	err = req.validateGrantTypes()
	if err != nil {
		return err
	}
	err = req.validateRedirectUris()
	if err != nil {
		return err
	}
	err = req.validateResponseTypes()
	if err != nil {
		return err
	}
	return nil
}

func (req *ClientRegistrationRequest) validateResponseTypes() error {
	if len(req.ResponseTypes) == 0 {
		return nil
	}
	for _, responseType := range req.ResponseTypes {
		if responseType != "code" {
			return fmt.Errorf("%w, received: %s", ErrInvalidResponseType, responseType)
		}
	}
	return nil
}
func (req *ClientRegistrationRequest) validateRedirectUris() error {
	if len(req.RedirectUris) == 0 {
		return fmt.Errorf("%w, redirect_uris was empty", ErrInvalidRedirectUri)
	}
	// TODO: allow for multiple redirect uris
	// , note that the reason for this is that go-oauth2
	// does not support multiple redirect uris
	// https://github.com/go-oauth2/oauth2/issues/257
	if len(req.RedirectUris) > 1 {
		return fmt.Errorf("%w, only one redirect_uri is supported", ErrInvalidRedirectUri)
	}
	for _, redirectUri := range req.RedirectUris {
		parsed, err := url.Parse(redirectUri)
		if err != nil {
			return fmt.Errorf("%w: %s: %w", ErrInvalidRedirectUri, redirectUri, err)
		}
		if parsed.Scheme != "https" {
			if isLoopback(parsed.Hostname()) {
				continue
			}
			return fmt.Errorf("%w: %s", ErrInvalidRedirectUri, redirectUri)
		}
	}
	return nil
}

func (req *ClientRegistrationRequest) validateTokenEndpointAuthMethod() error {
	if req.TokenEndpointAuthMethod == nil {
		return nil
	}
	if *req.TokenEndpointAuthMethod != "client_secret_post" {
		return fmt.Errorf("%w, received: %s", ErrInvalidTokenAuthMethod, *req.TokenEndpointAuthMethod)
	}
	return nil
}

func (req *ClientRegistrationRequest) validateGrantTypes() error {
	if len(req.GrantTypes) == 0 {
		return ErrInvalidGrantType
	}
	for _, grantType := range req.GrantTypes {
		if grantType != "authorization_code" {
			return fmt.Errorf("%w, received: %s", ErrInvalidGrantType, grantType)
		}
	}
	return nil
}

type ClientRegistrationResponse struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func (c *registeredClients) registerClient(req *ClientRegistrationRequest) (*oauthClient, error) {
	if err := req.validate(); err != nil {
		return nil, err
	}

	client := oauthClient{
		clientId:     uuid.NewString(),
		clientSecret: uuid.NewString(),
		redirectUris: req.RedirectUris,
	}

	c.Clients.Store(client.clientId, &client)
	return &client, nil
}

func isLoopback(hostname string) bool {
	if hostname == "localhost" {
		// shortcut for well-known loopback hostname
		return true
	}

	ipAddr := net.ParseIP(hostname)
	if ipAddr != nil {
		return ipAddr.IsLoopback()
	}

	resolvedIps, err := net.LookupIP(hostname)
	if err != nil {

		return false
	}

	for _, ip := range resolvedIps {
		if !ip.IsLoopback() {
			return false
		}
	}
	return true
}
