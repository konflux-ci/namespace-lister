package main

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/apis/apiserver"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/authenticatorfactory"
	authenticationv1 "k8s.io/client-go/kubernetes/typed/authentication/v1"
	"k8s.io/client-go/rest"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

func New(cfg *rest.Config) (authenticator.Request, *spec.SecurityDefinitions, error) {
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
