package cache_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/konflux-ci/namespace-lister/pkg/auth/cache"
	"github.com/konflux-ci/namespace-lister/pkg/auth/cache/mocks"
)

var (
	userSubject = rbacv1.Subject{
		Kind:     rbacv1.UserKind,
		APIGroup: rbacv1.SchemeGroupVersion.Group,
		Name:     "myuser",
	}

	groupSubject = rbacv1.Subject{
		Kind:     rbacv1.GroupKind,
		APIGroup: rbacv1.SchemeGroupVersion.Group,
		Name:     "mygroup",
	}

	serviceAccountSubject = rbacv1.Subject{
		Kind:      rbacv1.ServiceAccountKind,
		Name:      "myserviceaccount",
		Namespace: "mynamespace",
	}

	namespaces = []corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "myns",
				Labels:      map[string]string{"key": "value"},
				Annotations: map[string]string{"key": "value"},
			},
		},
	}

	expectedNamespacesUserAccess = []corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "myns",
				Labels: map[string]string{
					cache.VirtualLabelKeyAccess: "user",
					"key":                       "value",
				},
				Annotations: map[string]string{
					cache.VirtualAnnotationKeySubjectName: userSubject.Name,
					"key":                                 "value",
				},
			},
		},
	}

	expectedNamespacesServiceAccountAccess = []corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "myns",
				Labels: map[string]string{
					cache.VirtualLabelKeyAccess: "serviceaccount",
					"key":                       "value",
				},
				Annotations: map[string]string{
					cache.VirtualAnnotationKeySubjectName:      serviceAccountSubject.Name,
					cache.VirtualAnnotationKeySubjectNamespace: serviceAccountSubject.Namespace,
					"key": "value",
				},
			},
		},
	}

	expectedNamespacesGroupAccess = []corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "myns",
				Labels: map[string]string{
					cache.VirtualLabelKeyAccess: "group",
					"key":                       "value",
				},
				Annotations: map[string]string{
					cache.VirtualAnnotationKeySubjectName: groupSubject.Name,
					"key":                                 "value",
				},
			},
		},
	}
)

var _ = Describe("SynchronizedAccessCache", func() {
	var ctrl *gomock.Controller
	var subjectLocator *mocks.MockSubjectLocator

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		subjectLocator = mocks.NewMockSubjectLocator(ctrl)
	})

	It("will timeout on long running synch", func(ctx context.Context) {
		// given
		namespaceLister := mocks.NewMockClientReader(ctrl)
		namespaceLister.EXPECT().
			List(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, nn *corev1.NamespaceList, opts ...client.ListOption) error {
				<-ctx.Done()
				return ctx.Err()
			}).
			Times(1)
		nsc := cache.NewSynchronizedAccessCache(subjectLocator, namespaceLister, cache.CacheSynchronizerOptions{
			SynchTimeout: 1 * time.Second,
		})

		// when
		Expect(nsc.Synch(ctx)).
			// then
			To(MatchError("context deadline exceeded"))
	})

	It("will return empty data if timed-out", func(ctx context.Context) {
		// given
		namespaceLister := mocks.NewMockClientReader(ctrl)
		namespaceLister.EXPECT().
			List(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, nn *corev1.NamespaceList, opts ...client.ListOption) error {
				(&corev1.NamespaceList{
					Items: []corev1.Namespace{
						{ObjectMeta: metav1.ObjectMeta{Name: "myns-1"}},
						{ObjectMeta: metav1.ObjectMeta{Name: "myns-2"}},
						{ObjectMeta: metav1.ObjectMeta{Name: "myns-3"}},
					},
				}).DeepCopyInto(nn)
				return nil
			}).
			Times(1)

		subjectLocator.EXPECT().
			AllowedSubjects(gomock.Any(), gomock.Any()).
			// reply fast the first time
			Return([]rbacv1.Subject{userSubject}, nil).
			DoAndReturn(func(ctx context.Context, attributes authorizer.Attributes) ([]rbacv1.Subject, error) {
				// cause a timeout the second time
				time.Sleep(100 * time.Millisecond)
				return []rbacv1.Subject{userSubject}, nil
			})

		nsc := cache.NewSynchronizedAccessCache(subjectLocator, namespaceLister, cache.CacheSynchronizerOptions{
			SynchTimeout: 50 * time.Millisecond,
		})

		// when
		Expect(nsc.Synch(ctx)).
			// then
			To(MatchError("context deadline exceeded"))
	})

	It("can not run synch twice", func(ctx context.Context) {
		// given
		namespaceLister := mocks.NewMockClientReader(ctrl)
		namespaceLister.EXPECT().
			List(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, nn *corev1.NamespaceList, opts ...client.ListOption) error {
				time.Sleep(5 * time.Second)
				return nil
			}).
			Times(1)
		nsc := cache.NewSynchronizedAccessCache(subjectLocator, namespaceLister, cache.CacheSynchronizerOptions{})

		// when
		go func() { _ = nsc.Synch(ctx) }()
		time.Sleep(1 * time.Second)

		// then
		Expect(nsc.Synch(ctx)).To(MatchError(cache.ErrSynchAlreadyRunning))
	})

	It("restocks cache with empty list", func(ctx context.Context) {
		namespaceLister := mocks.NewMockClientReader(ctrl)
		namespaceLister.EXPECT().
			List(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, nn *corev1.NamespaceList, opts ...client.ListOption) error {
				(&corev1.NamespaceList{Items: namespaces}).DeepCopyInto(nn)
				return nil
			}).
			Times(1)
		subjectLocator.EXPECT().
			AllowedSubjects(gomock.Any(), gomock.Any()).
			Return([]rbacv1.Subject{}, nil).
			Times(1)

		nsc := cache.NewSynchronizedAccessCache(subjectLocator, namespaceLister, cache.CacheSynchronizerOptions{})

		Expect(nsc.Synch(ctx)).ToNot(HaveOccurred())
		Expect(nsc.AccessCache.List(userSubject)).To(BeEmpty())
	})

	It("matches user after synch", func(ctx context.Context) {
		namespaceLister := mocks.NewMockClientReader(ctrl)
		namespaceLister.EXPECT().
			List(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, nn *corev1.NamespaceList, opts ...client.ListOption) error {
				(&corev1.NamespaceList{Items: namespaces}).DeepCopyInto(nn)
				return nil
			}).
			Times(1)
		subjectLocator.EXPECT().
			AllowedSubjects(gomock.Any(), gomock.Any()).
			Return([]rbacv1.Subject{userSubject}, nil).
			Times(1)

		nsc := cache.NewSynchronizedAccessCache(subjectLocator, namespaceLister, cache.CacheSynchronizerOptions{})

		Expect(nsc.Synch(ctx)).ToNot(HaveOccurred())
		Expect(nsc.AccessCache.List(userSubject)).To(ConsistOf(expectedNamespacesUserAccess))
	})

	It("matches ServiceAccount after synch", func(ctx context.Context) {
		namespaceLister := mocks.NewMockClientReader(ctrl)
		namespaceLister.EXPECT().
			List(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, nn *corev1.NamespaceList, opts ...client.ListOption) error {
				(&corev1.NamespaceList{Items: namespaces}).DeepCopyInto(nn)
				return nil
			}).
			Times(1)
		subjectLocator.EXPECT().
			AllowedSubjects(gomock.Any(), gomock.Any()).
			Return([]rbacv1.Subject{serviceAccountSubject}, nil).
			Times(1)

		nsc := cache.NewSynchronizedAccessCache(subjectLocator, namespaceLister, cache.CacheSynchronizerOptions{})

		Expect(nsc.Synch(ctx)).ToNot(HaveOccurred())
		Expect(nsc.AccessCache.List(serviceAccountSubject)).To(ConsistOf(expectedNamespacesServiceAccountAccess))
	})

	It("does cache Groups", func(ctx context.Context) {
		namespaceLister := mocks.NewMockClientReader(ctrl)
		namespaceLister.EXPECT().
			List(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, nn *corev1.NamespaceList, opts ...client.ListOption) error {
				(&corev1.NamespaceList{Items: namespaces}).DeepCopyInto(nn)
				return nil
			}).
			Times(1)
		subjectLocator.EXPECT().
			AllowedSubjects(gomock.Any(), gomock.Any()).
			Return([]rbacv1.Subject{groupSubject}, nil).
			Times(1)

		nsc := cache.NewSynchronizedAccessCache(subjectLocator, namespaceLister, cache.CacheSynchronizerOptions{})

		Expect(nsc.Synch(ctx)).ToNot(HaveOccurred())
		Expect(nsc.AccessCache.List(groupSubject)).To(ConsistOf(expectedNamespacesGroupAccess))
	})
})

var _ = DescribeTable("duplicate results", func(ctx context.Context, sr *mocks.MockStaticRoles) {
	ctrl := gomock.NewController(GinkgoT())
	namespaceLister := mocks.NewMockClientReader(ctrl)
	namespaceLister.EXPECT().
		List(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, nn *corev1.NamespaceList, opts ...client.ListOption) error {
			(&corev1.NamespaceList{Items: namespaces}).DeepCopyInto(nn)
			return nil
		}).
		Times(1)

	realSubjectLocator := rbac.NewSubjectAccessEvaluator(sr, sr, sr, sr, "")

	nsc := cache.NewSynchronizedAccessCache(realSubjectLocator, namespaceLister, cache.CacheSynchronizerOptions{})

	Expect(nsc.Synch(ctx)).To(Succeed())
	Expect(nsc.AccessCache.List(userSubject)).To(ConsistOf(expectedNamespacesUserAccess))
},
	Entry("does not produce duplicates with multiple RoleBindings to access ClusterRole", &mocks.MockStaticRoles{
		ClusterRoles: []*rbacv1.ClusterRole{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "user-access-clusterrole",
				},
				Rules: []rbacv1.PolicyRule{
					{Verbs: []string{"get"}, Resources: []string{"namespaces"}, APIGroups: []string{""}},
				},
			},
		},
		RoleBindings: []*rbacv1.RoleBinding{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user-access-role-bindings-1",
					Namespace: namespaces[0].Name,
				},
				RoleRef:  rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "ClusterRole", Name: "user-access-clusterrole"},
				Subjects: []rbacv1.Subject{userSubject},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user-access-role-bindings-2",
					Namespace: namespaces[0].Name,
				},
				RoleRef:  rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "ClusterRole", Name: "user-access-clusterrole"},
				Subjects: []rbacv1.Subject{userSubject},
			},
		},
	}),
	Entry("does not produce duplicates with multiple RoleBindings to access Role", &mocks.MockStaticRoles{
		Roles: []*rbacv1.Role{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user-access-role",
					Namespace: namespaces[0].Name,
				},
				Rules: []rbacv1.PolicyRule{
					{Verbs: []string{"get"}, Resources: []string{"namespaces"}, APIGroups: []string{""}, ResourceNames: []string{namespaces[0].Name}},
				},
			},
		},
		RoleBindings: []*rbacv1.RoleBinding{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user-access-role-bindings-1",
					Namespace: namespaces[0].Name,
				},
				RoleRef:  rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "Role", Name: "user-access-role"},
				Subjects: []rbacv1.Subject{userSubject},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user-access-role-bindings-2",
					Namespace: namespaces[0].Name,
				},
				RoleRef:  rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "Role", Name: "user-access-role"},
				Subjects: []rbacv1.Subject{userSubject},
			},
		},
	}),
)
