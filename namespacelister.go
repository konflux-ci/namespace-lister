package main

import (
	"context"
	"log/slog"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ NamespaceLister = &namespaceLister{}

type NamespaceLister interface {
	ListNamespaces(ctx context.Context, username string) (*corev1.NamespaceList, error)
}

type namespaceLister struct {
	client.Reader

	authorizer *rbac.RBACAuthorizer
	l          *slog.Logger
}

func NewNamespaceLister(reader client.Reader, authorizer *rbac.RBACAuthorizer, l *slog.Logger) NamespaceLister {
	return &namespaceLister{
		Reader:     reader,
		authorizer: authorizer,
		l:          l,
	}
}

func (c *namespaceLister) ListNamespaces(ctx context.Context, username string) (*corev1.NamespaceList, error) {
	// list role bindings
	nn := corev1.NamespaceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NamespaceList",
			APIVersion: "",
		},
	}
	if err := c.List(ctx, &nn); err != nil {
		return nil, err
	}

	rnn := []corev1.Namespace{}
	for _, ns := range nn.Items {
		d, _, err := c.authorizer.Authorize(ctx, authorizer.AttributesRecord{
			User:            &user.DefaultInfo{Name: username},
			Verb:            "get",
			Resource:        "namespaces",
			APIGroup:        "",
			APIVersion:      "v1",
			Name:            ns.Name,
			Namespace:       ns.Name,
			ResourceRequest: true,
		})
		if err != nil {
			return nil, err
		}

		c.l.Info("evaluated user access to namespace", "namespace", ns.Name, "user", username, "decision", d)
		if d == authorizer.DecisionAllow {
			rnn = append(rnn, ns)
		}
	}
	nn.Items = rnn

	return &nn, nil
}
