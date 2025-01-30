package main_test

import (
	"k8s.io/client-go/rest"

	namespacelister "github.com/konflux-ci/namespace-lister"
)

//go:generate mockgen -source=interfaces_test.go -destination=mocks/rest_interface.go -package=mocks

type FakeInterface interface {
	rest.Interface
}

type FakeSubjectNamespacesLister interface {
	namespacelister.SubjectNamespacesLister
}
