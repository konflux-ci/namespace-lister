package main

import (
	"errors"
	"net/http"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/apis/apiserver"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/authenticatorfactory"
	"k8s.io/apiserver/pkg/authentication/user"
	authenticationv1 "k8s.io/client-go/kubernetes/typed/authentication/v1"
	"k8s.io/client-go/rest"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

// Authenticator authenticates requests.
// If Header authentication is enabled, it will check the value in the request Header accordingly.
// If the value in the header was provided, it assumes a proxy already authenticated the request.
//
// If header authentication is disabled or the header is not set in the request,
// the request is authenticated by the DelegatingAuthenticator configured with
// disabled anonymous access and enabled TokenAccessReview. This means it will look for a JWT
// Token in the request and ask the APIServer to authenticate it.
// APIServer replies are cached for a short time.
type Authenticator struct {
	usernameHeader string
	next           authenticator.Request
}

// AuthenticateRequest authenticates a request by checking the username header
// and/or validating the token through the APIServer.
func (a *Authenticator) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) {
	if a.usernameHeader != "" {
		if username := req.Header.Get(a.usernameHeader); username != "" {
			// TODO: parse User, Group, ServiceAccount.
			// Look into Kubernetes libs before implementing it
			return &authenticator.Response{
				User: &user.DefaultInfo{
					Name: username,
				},
			}, true, nil
		}
	}

	return a.next.AuthenticateRequest(req)
}

// AuthenticatorOptions allows to configure the Authenticator
type AuthenticatorOptions struct {
	Client rest.Interface
	Config *rest.Config
	Header string
}

// NewAuthenticator builds a new Authenticator
func NewAuthenticator(opts AuthenticatorOptions) (authenticator.Request, error) {
	ar, _, err := newTokenReviewAuthenticatorWithOpts(&opts)
	if err != nil {
		return nil, err
	}

	return &Authenticator{
		usernameHeader: opts.Header,
		next:           ar,
	}, nil
}

// NewTokenReviewAuthenticatorWithClient builds a TokenReviewAuthenticator from a kubernetes client
func NewTokenReviewAuthenticatorWithClient(c rest.Interface) (authenticator.Request, *spec.SecurityDefinitions, error) {
	tokenAccessReviewClient := authenticationv1.New(c)
	return newTokenReviewAuthenticator(tokenAccessReviewClient)
}

// NewTokenReviewAuthenticatorWithConfig builds a TokenReviewAuthenticator from a kubernetes client configuration
func NewTokenReviewAuthenticatorWithConfig(cfg *rest.Config) (authenticator.Request, *spec.SecurityDefinitions, error) {
	cfg = rest.CopyConfig(cfg)
	tokenAccessReviewClient := authenticationv1.NewForConfigOrDie(cfg)
	return newTokenReviewAuthenticator(tokenAccessReviewClient)
}

func newTokenReviewAuthenticator(authenticationClient *authenticationv1.AuthenticationV1Client) (authenticator.Request, *spec.SecurityDefinitions, error) {
	authCfg := authenticatorfactory.DelegatingAuthenticatorConfig{
		Anonymous:                &apiserver.AnonymousAuthConfig{Enabled: false},
		TokenAccessReviewClient:  authenticationClient,
		TokenAccessReviewTimeout: 1 * time.Minute,
		WebhookRetryBackoff:      &wait.Backoff{Duration: 2 * time.Second, Cap: 2 * time.Minute, Steps: 100, Factor: 2, Jitter: 2},
		CacheTTL:                 5 * time.Minute,
	}
	return authCfg.New()
}

func newTokenReviewAuthenticatorWithOpts(opts *AuthenticatorOptions) (authenticator.Request, *spec.SecurityDefinitions, error) {
	switch {
	case opts.Client != nil:
		return NewTokenReviewAuthenticatorWithClient(opts.Client)
	case opts.Config != nil:
		return NewTokenReviewAuthenticatorWithConfig(opts.Config)
	default:
		return nil, nil, errors.New("one among client and config is required to build the TokenRevierAuthenticator")
	}
}

// GetUsernameHeaderFromEnv retrieves from environment variable the name of the header
// to use when authenticating requests by username header
func GetUsernameHeaderFromEnv() string {
	return os.Getenv(EnvUsernameHeader)
}
