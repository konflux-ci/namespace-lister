package main_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/gomega"

	namespacelister "github.com/konflux-ci/namespace-lister"
	"github.com/konflux-ci/namespace-lister/mocks"
)

var _ = Describe("Subjectnamespaceslister", func() {
	var subjectNamespacesLister *mocks.MockFakeSubjectNamespacesLister
	enn := []corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "myns",
				Labels:      map[string]string{"key": "value"},
				Annotations: map[string]string{"key": "value"},
			},
		},
	}

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		defer ctrl.Finish()

		subjectNamespacesLister = mocks.NewMockFakeSubjectNamespacesLister(ctrl)
	})

	It("parses service account", func(ctx context.Context) {
		// set expectation
		subjectNamespacesLister.EXPECT().
			List(
				ctx,
				rbacv1.Subject{
					Kind:      "ServiceAccount",
					Name:      "myserviceaccount",
					Namespace: "mynamespace",
				},
			).
			Return(enn).
			Times(1)

		// given
		nl := namespacelister.NewNamespaceListerForSubject(subjectNamespacesLister)

		// when
		Expect(nl.ListNamespaces(ctx, "system:serviceaccount:mynamespace:myserviceaccount")).
			// then
			To(BeEquivalentTo(&corev1.NamespaceList{
				TypeMeta: metav1.TypeMeta{
					Kind:       "NamespaceList",
					APIVersion: "v1",
				},
				Items: enn,
			}))
	})

	It("parses user", func(ctx context.Context) {
		// set expectation
		subjectNamespacesLister.EXPECT().
			List(
				ctx,
				rbacv1.Subject{
					APIGroup: rbacv1.GroupName,
					Kind:     "User",
					Name:     "myuser",
				},
			).
			Return(enn).
			Times(1)

		// given
		nl := namespacelister.NewNamespaceListerForSubject(subjectNamespacesLister)

		// when
		Expect(nl.ListNamespaces(ctx, "myuser")).
			// then
			To(BeEquivalentTo(&corev1.NamespaceList{
				TypeMeta: metav1.TypeMeta{
					Kind:       "NamespaceList",
					APIVersion: "v1",
				},
				Items: enn,
			}))
	})
})
