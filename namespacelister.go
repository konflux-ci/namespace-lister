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
	// list all namespaces
	nn := corev1.NamespaceList{
		TypeMeta: metav1.TypeMeta{
			// even though `kubectl get namespaces -o yaml` is showing `kind: List`
			// the plain response from the APIServer is using `kind: NamespaceList`.
			// Use `kubectl get namespaces -v9` to inspect the APIServer plain response.
			Kind:       "NamespaceList",
			APIVersion: corev1.SchemeGroupVersion.Version,
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
			APIGroup:        corev1.GroupName,
			APIVersion:      corev1.SchemeGroupVersion.Version,
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
