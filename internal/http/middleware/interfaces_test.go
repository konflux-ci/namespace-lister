package middleware_test

import "k8s.io/apiserver/pkg/authentication/authenticator"

//go:generate mockgen -source=interfaces_test.go -destination=mocks/middleware_interface.go -package=mocks

type Request interface {
	authenticator.Request
}
