package streamable_http

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/strowk/foxy-contexts/pkg/auth"
)

// Oauth2.1 for Authorization support
//
// Some facts about implementation here:
// Only "code" respnose type is supported, due to Oauth2.1 removal of implicit flow:
// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-12#name-removal-of-the-oauth-20-imp
//
// Only "form_post" response mode supported because this protects from leaking to third parties
// in case if client was not confidential.
// See more in: https://openid.net/specs/oauth-v2-multiple-response-types-1_0.html#Security
//
// Only "authorization_code", "refresh_token" and "client_credentials" could be supported
// in the future, but for now only "authorization_code" is supported. Oauth2.1 only defines
// these three flows in https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-12#section-3.2.2
//
// Only "client_secret_post" is supported for "token_endpoint_auth_method" due to Oauth2.1
// recommending this due to interoperability reasons because HTTP Basic is often incorrectly implemented:
// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-12#name-client-secret
//
// Only "RS256" is supported for "token_endpoint_auth_signing_alg_values_supported"
// as this is the only one which servers SHOULD support according to "OAuth 2.0 Authorization Server Metadata":
// https://datatracker.ietf.org/doc/html/rfc8414#section-2
//
// Only "S256" is supported for "code_challenge_method" because it is mandatory to implement for servers
// according to Oauth2.1: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-12#section-4.1.1

const (
	AuthorizePath = "/authorize"
	TokenPath     = "/token"
	RegisterPath  = "/register"
	JWKSPath      = "/jwks"
)

var (
	ErrWrongMinTLSVersion = fmt.Errorf("Wrong minimum TLS version - authorization requires TLS 1.3 or higher")
)

func (s *streamableHttpTransport) pathExcludedFromAuth(path string) bool {
	for _, excluded := range s.authorization.ExcludedPaths() {
		if path == excluded {
			return true
		}
	}
	return false
}

func (s *streamableHttpTransport) getAuthorizationMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Method == "OPTIONS" {
				// Preflight request, no need to check authorization
				return next(c)
			}
			if c.Request().URL.Path == AuthorizePath ||
				c.Request().URL.Path == TokenPath ||
				c.Request().URL.Path == RegisterPath ||
				c.Request().URL.Path == auth.WellKnownDiscoveryPath ||
				s.pathExcludedFromAuth(c.Request().URL.Path) {
				// Authorization endpoints, no need to check authorization
				return next(c)
			}
			if err := s.checkAuthorization(c); err != nil {
				return err
			}
			return next(c)
		}
	}
}

type authUserIdKeyType struct{}

var authUserIdKey authUserIdKeyType = struct{}{}

func (s *streamableHttpTransport) checkAuthorization(c echo.Context) error {
	if !s.authorizationRequired {
		return nil
		// TODO: even when not required, attempt to resolve authorization
		// to set user id to context
	}
	authorizationHeader := c.Request().Header.Get("Authorization")
	if authorizationHeader == "" {
		return echo.NewHTTPError(401, "Authorization header is required")
	}
	if !s.isAuthorizationValid(authorizationHeader) {
		return echo.NewHTTPError(401, "Invalid authorization")
	}

	userId, err := s.authorization.ValidateToken(c.Request().Context(), strings.TrimPrefix(authorizationHeader, "Bearer "))

	// save user id in request context for later use
	c.SetRequest(c.Request().WithContext(
		context.WithValue(c.Request().Context(), authUserIdKey, userId),
	))

	return err
}

func (s *streamableHttpTransport) isAuthorizationValid(header string) bool {
	header = strings.TrimSpace(header)
	if !strings.HasPrefix(header, "Bearer ") {
		return false
	}
	token := strings.TrimPrefix(header, "Bearer ")
	return token != ""
}

func (s *streamableHttpTransport) AsAuthorizeable() auth.AuthorizeableTransport {
	return s
}

func (s *streamableHttpTransport) PlugInAuthorization(authorization auth.Authorization) error {

	s.authorization = authorization
	s.authorizationRequired = authorization.IsRequired()

	s.configureAuthMiddleware()

	if s.tlsConfig != nil && s.tlsConfig.MinVersion != tls.VersionTLS13 {
		return ErrWrongMinTLSVersion
	}

	s.e.GET(auth.WellKnownDiscoveryPath, func(c echo.Context) error {
		// AuthorizationServerMetadata as defined in RFC 8414:
		// https://datatracker.ietf.org/doc/html/rfc8414#section-2
		type AuthorizationServerMetadata struct {
			Issuer                                     string   `json:"issuer"`
			AuthorizationEndpoint                      string   `json:"authorization_endpoint"`
			TokenEndpoint                              string   `json:"token_endpoint"`
			JWKSUri                                    string   `json:"jwks_uri"`
			RegistrationEndpoint                       string   `json:"registration_endpoint"`
			ScopesSupported                            []string `json:"scopes_supported"`
			ResponseTypesSupported                     []string `json:"response_types_supported"`
			ResponseModesSupported                     []string `json:"response_modes_supported"`
			GrantTypesSupported                        []string `json:"grant_types_supported"`
			TokenEndpointAuthMethodsSupported          []string `json:"token_endpoint_auth_methods_supported"`
			TokenEndpointAuthSigningAlgValuesSupported []string `json:"token_endpoint_auth_signing_alg_values_supported"`
			CodeChallengeMethodsSupported              []string `json:"code_challenge_methods_supported"`
		}

		return c.JSON(200, AuthorizationServerMetadata{
			Issuer: fmt.Sprintf("%s://%s%s", s.authScheme, s.authHost, s.path),

			// TODO: figure out if it makes sense to use JWK at all
			// JWKSUri: fmt.Sprintf("https://%s%s", s.authHost, JWKSPath),

			AuthorizationEndpoint: fmt.Sprintf("%s://%s%s", s.authScheme, s.authHost, AuthorizePath),
			TokenEndpoint:         fmt.Sprintf("%s://%s%s", s.authScheme, s.authHost, TokenPath),
			RegistrationEndpoint:  fmt.Sprintf("%s://%s%s", s.authScheme, s.authHost, RegisterPath),

			ScopesSupported:                            []string{},
			ResponseTypesSupported:                     auth.ResponseTypesSupported,
			ResponseModesSupported:                     auth.ResponseModesSupported,
			GrantTypesSupported:                        auth.GrantTypesSupported,
			TokenEndpointAuthMethodsSupported:          auth.TokenEndpointAuthMethodsSupported,
			TokenEndpointAuthSigningAlgValuesSupported: auth.TokenEndpointAuthSigningAlgValuesSupported,
			CodeChallengeMethodsSupported:              auth.CodeChallengeMethodsSupported,
		})
	})

	s.e.POST(RegisterPath, func(c echo.Context) error {
		if c.Request().Header.Get("Content-Type") != "application/json" {
			return echo.NewHTTPError(400, "Content-Type must be application/json")
		}
		var req auth.ClientRegistrationRequest
		if err := c.Bind(&req); err != nil {
			return echo.NewHTTPError(400, "Invalid request body")
		}
		res, err := authorization.RegisterClient(c.Request().Context(), &req)
		if err != nil {
			return echo.NewHTTPError(400, err.Error())
		}
		return c.JSON(200, res)
	})

	s.e.GET(AuthorizePath, func(c echo.Context) error {
		err := authorization.HandleAuthorizeRequest(c.Request().Context(), c.Response(), c.Request())
		if err != nil {
			return echo.NewHTTPError(400, err.Error())
		}
		return nil
	})

	s.e.POST(TokenPath, func(c echo.Context) error {
		err := authorization.HandleTokenRequest(c.Request().Context(), c.Response(), c.Request())
		if err != nil {
			return echo.NewHTTPError(400, err.Error())
		}
		return nil
	})

	return nil
}
