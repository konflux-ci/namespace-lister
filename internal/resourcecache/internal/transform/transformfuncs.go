package transform

import (
	"fmt"
	"slices"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
)

func MergeTransformFunc(ff ...toolscache.TransformFunc) toolscache.TransformFunc {
	return func(i any) (any, error) {
		var err error

		for _, f := range ff {
			if i, err = f(i); err != nil {
				return nil, err
			}
		}
		return i, nil
	}
}

func TrimAnnotations() toolscache.TransformFunc {
	return func(in any) (any, error) {
		if obj, err := meta.Accessor(in); err == nil && obj.GetAnnotations() != nil {
			obj.SetAnnotations(nil)
		}

		return in, nil
	}
}

func TrimRole() toolscache.TransformFunc {
	return MergeTransformFunc(
		cache.TransformStripManagedFields(),
		TrimAnnotations(),
		func(i any) (any, error) {
			r, ok := i.(*rbacv1.Role)
			if !ok {
				return nil, fmt.Errorf("error caching Role: expected Role received %T", i)
			}

			r.Rules = filterNamespacesRelatedPolicyRules(r.Rules)
			if len(r.Rules) == 0 {
				return nil, nil
			}
			return r, nil
		},
	)
}

func TrimClusterRole() toolscache.TransformFunc {
	return MergeTransformFunc(
		cache.TransformStripManagedFields(),
		TrimAnnotations(),
		func(i any) (any, error) {
			cr, ok := i.(*rbacv1.ClusterRole)
			if !ok {
				return nil, fmt.Errorf("error caching ClusterRole: expected a ClusterRole received %T", i)
			}

			cr.Rules = filterNamespacesRelatedPolicyRules(cr.Rules)
			if len(cr.Rules) == 0 {
				return nil, nil
			}
			return cr, nil
		},
	)
}

func TrimNamespace() toolscache.TransformFunc {
	return MergeTransformFunc(
		cache.TransformStripManagedFields(),
		func(i any) (any, error) {
			ns, ok := i.(*corev1.Namespace)
			if !ok {
				return nil, fmt.Errorf("error caching Namespace: expected a Namespace received %T", i)
			}

			ns.Spec = corev1.NamespaceSpec{}
			ns.Status = corev1.NamespaceStatus{}
			return ns, nil
		})
}

var (
	TrimClusterRoleBinding = trimBinding
	TrimRoleBinding        = trimBinding
)

func trimBinding() toolscache.TransformFunc {
	return MergeTransformFunc(cache.TransformStripManagedFields(), TrimAnnotations())
}

func filterNamespacesRelatedPolicyRules(pp []rbacv1.PolicyRule) []rbacv1.PolicyRule {
	var fr []rbacv1.PolicyRule
	for _, r := range pp {
		if slices.Contains(r.APIGroups, "") &&
			slices.Contains(r.Resources, "namespaces") &&
			slices.Contains(r.Verbs, "get") {
			r.APIGroups = []string{""}
			r.Resources = []string{"namespaces"}
			r.Verbs = []string{"get"}
			fr = append(fr, r)
		}
	}
	return fr
}
