package resourcecache

import (
	"cmp"

	"github.com/konflux-ci/namespace-lister/internal/resourcecache/internal/transform"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NamespaceUseForAccessLabelKey       = "namespace-lister.konflux-ci.dev/use-for-access"
	NamespaceUseForAccessLabelValueTrue = "true"
)

var cachedObjects = []client.Object{
	&corev1.Namespace{},
	&rbacv1.RoleBinding{},
	&rbacv1.ClusterRole{},
	&rbacv1.ClusterRoleBinding{},
	&rbacv1.Role{},
}

func build(cfg *Config) (cache.Cache, error) {
	// build scheme
	s, err := buildScheme()
	if err != nil {
		return nil, err
	}

	// build embedded cache options
	o := buildCacheOptions(s, cfg.NamespacesLabelSelector)

	// build cache
	return cache.New(cfg.RestConfig, o)
}

func buildScheme() (*runtime.Scheme, error) {
	s := runtime.NewScheme()
	if err := cmp.Or(
		corev1.AddToScheme(s),
		rbacv1.AddToScheme(s),
	); err != nil {
		return nil, err
	}

	return s, nil
}

func buildCacheOptions(s *runtime.Scheme, namespaceSelector labels.Selector) cache.Options {
	return cache.Options{
		Scheme:                       s,
		DefaultUnsafeDisableDeepCopy: ptr.To(true),
		ReaderFailOnMissingInformer:  true,
		ByObject:                     byObjectTransformers(namespaceSelector),
	}
}

func byObjectTransformers(namespaceSelector labels.Selector) map[client.Object]cache.ByObject {
	return map[client.Object]cache.ByObject{
		&corev1.Namespace{}: {
			Label:     namespaceSelector,
			Transform: transform.TrimNamespace(),
		},
		&rbacv1.Role{}: {
			Transform: transform.TrimRole(),
		},
		&rbacv1.RoleBinding{}: {
			Transform: transform.TrimRoleBinding(),
		},
		&rbacv1.ClusterRole{}: {
			Transform: transform.TrimClusterRole(),
		},
		&rbacv1.ClusterRoleBinding{}: {
			Transform: transform.TrimClusterRoleBinding(),
			Label:     labels.SelectorFromSet(labels.Set{NamespaceUseForAccessLabelKey: NamespaceUseForAccessLabelValueTrue})},
	}
}
