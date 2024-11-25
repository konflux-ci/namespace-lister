package main

import (
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

type Authenticator struct {
	usernameHeader string
	next           authenticator.Request
}

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

func New(cfg *rest.Config) (authenticator.Request, error) {
	ar, _, err := NewTokenReviewAuthenticator(cfg)
	if err != nil {
		return nil, err
	}

	return &Authenticator{
		usernameHeader: getUsernameHeaderFromEnv(),
		next:           ar,
	}, nil
}

func NewTokenReviewAuthenticator(cfg *rest.Config) (authenticator.Request, *spec.SecurityDefinitions, error) {
	cfg = rest.CopyConfig(cfg)

	authCfg := authenticatorfactory.DelegatingAuthenticatorConfig{
		Anonymous:                &apiserver.AnonymousAuthConfig{Enabled: false},
		TokenAccessReviewClient:  authenticationv1.NewForConfigOrDie(cfg),
		TokenAccessReviewTimeout: 1 * time.Minute,
		WebhookRetryBackoff:      &wait.Backoff{Duration: 2 * time.Second, Cap: 2 * time.Minute, Steps: 100, Factor: 2, Jitter: 2},
		CacheTTL:                 5 * time.Minute,
	}
	return authCfg.New()
}

func getUsernameHeaderFromEnv() string {
	return os.Getenv(EnvUsernameHeader)
}
