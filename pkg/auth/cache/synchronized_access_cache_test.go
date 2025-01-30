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
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/konflux-ci/namespace-lister/pkg/auth/cache"
	"github.com/konflux-ci/namespace-lister/pkg/auth/cache/mocks"
)

var _ = Describe("SynchronizedAccessCache", func() {
	var ctrl *gomock.Controller
	var subjectLocator *mocks.MockSubjectLocator
	var namespaceListerBuilder fake.ClientBuilder

	userSubject := rbacv1.Subject{
		Kind:     "User",
		APIGroup: rbacv1.SchemeGroupVersion.Group,
		Name:     "myuser",
	}

	serviceAccountSubject := rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      "myserviceaccount",
		Namespace: "mynamespace",
	}

	namespaces := []corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "myns",
				Labels:      map[string]string{"key": "value"},
				Annotations: map[string]string{"key": "value"},
			},
		},
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		subjectLocator = mocks.NewMockSubjectLocator(ctrl)
		s := runtime.NewScheme()
		utilruntime.Must(corev1.AddToScheme(s))
		namespaceListerBuilder.WithScheme(s)
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
		Expect(nsc.Synch(ctx)).To(MatchError(cache.SynchAlreadyRunningErr))
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
			AllowedSubjects(gomock.Any()).
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
			AllowedSubjects(gomock.Any()).
			Return([]rbacv1.Subject{userSubject}, nil).
			Times(1)

		nsc := cache.NewSynchronizedAccessCache(subjectLocator, namespaceLister, cache.CacheSynchronizerOptions{})

		Expect(nsc.Synch(ctx)).ToNot(HaveOccurred())
		Expect(nsc.AccessCache.List(userSubject)).To(BeEquivalentTo(namespaces))
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
			AllowedSubjects(gomock.Any()).
			Return([]rbacv1.Subject{serviceAccountSubject}, nil).
			Times(1)

		nsc := cache.NewSynchronizedAccessCache(subjectLocator, namespaceLister, cache.CacheSynchronizerOptions{})

		Expect(nsc.Synch(ctx)).ToNot(HaveOccurred())
		Expect(nsc.AccessCache.List(serviceAccountSubject)).To(BeEquivalentTo(namespaces))
	})
})
