package auth

// WithAuthRequired is an option that sets the authorization as required
// for the server. When true, the server will respond to all requests without
// authorization with an unauthorized error (401 HTTP status code).
//
// When Oauth2 is setup, it is required by default unless disabled by this option.
func WithAuthRequired(required bool) Oauth2AuthorizationOption {
	return func(o *oauth2Auth) {
		o.required = required
	}
}

func WithAuthorizationHandler(handler AuthorizationHandler) Oauth2AuthorizationOption {
	return func(o *oauth2Auth) {
		o.authorizationHandler = handler
	}
}

func WithExcludedPaths(paths ...string) Oauth2AuthorizationOption {
	return func(o *oauth2Auth) {
		o.excludedPaths = paths
	}
}
