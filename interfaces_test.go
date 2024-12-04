package main_test

import (
	"k8s.io/client-go/rest"
)

//go:generate mockgen -source=interfaces_test.go -destination=mocks/rest_interface.go -package=mocks
type FakeInterface interface {
	rest.Interface
}
