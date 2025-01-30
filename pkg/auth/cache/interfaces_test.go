package cache_test

import (
	"k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -source=interfaces_test.go -destination=mocks/mocks.go -package=mocks

type SubjectLocator interface {
	rbac.SubjectLocator
}

type ClientReader interface {
	client.Reader
}
