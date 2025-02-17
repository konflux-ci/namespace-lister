package main_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	namespacelister "github.com/konflux-ci/namespace-lister"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
)

type NamespaceListerMock func(ctx context.Context, username string) (*corev1.NamespaceList, error)

func (m NamespaceListerMock) ListNamespaces(ctx context.Context, username string) (*corev1.NamespaceList, error) {
	return m(ctx, username)
}

var _ = Describe("HttpHandlerList", func() {
	var request *http.Request

	BeforeEach(func(tctx context.Context) {
		var err error
		ctx := context.WithValue(tctx, namespacelister.ContextKeyUserDetails,
			&authenticator.Response{
				User: &user.DefaultInfo{
					Name: "myuser",
				},
			})
		request, err = http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		Expect(err).NotTo(HaveOccurred())
	})

	DescribeTable("retrieves list of namespaces", func(expected corev1.NamespaceList) {
		// given
		eb, err := json.Marshal(expected)
		if err != nil {
			panic(err)
		}
		lister := NamespaceListerMock(func(ctx context.Context, username string) (*corev1.NamespaceList, error) {
			return &expected, nil
		})
		handler := namespacelister.NewListNamespacesHandler(lister)

		w := httptest.NewRecorder()

		// when
		handler.ServeHTTP(w, request)

		// then
		Expect(w.Result()).NotTo(BeNil())
		Expect(w.Result().StatusCode).To(Equal(http.StatusOK))
		Expect(w.Result().Header.Get(namespacelister.HttpContentType)).To(Equal(namespacelister.HttpContentTypeApplication))
		Expect(io.ReadAll(w.Result().Body)).To(Equal(eb))
	},
		Entry("empty list", corev1.NamespaceList{}),
		Entry("non empty list", corev1.NamespaceList{
			Items: []corev1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "myns",
					},
				},
			},
		}),
	)

	DescribeTable("returns an error when lister returns an error", func(expectedErr error, expectedResponseStatus int) {
		// given
		lister := NamespaceListerMock(func(ctx context.Context, username string) (*corev1.NamespaceList, error) {
			return nil, expectedErr
		})
		handler := namespacelister.NewListNamespacesHandler(lister)

		w := httptest.NewRecorder()

		// when
		handler.ServeHTTP(w, request)

		// then
		Expect(w.Result()).NotTo(BeNil())
		Expect(w.Result().StatusCode).To(Equal(expectedResponseStatus))
		// Expect(w.Result().Header.Get(HttpContentType)).To(Equal(HttpContentTypeApplication))
		Expect(io.ReadAll(w.Result().Body)).To(BeEquivalentTo(expectedErr.Error()))
	},
		Entry("unhandled error", errors.New("unhandled error"), http.StatusInternalServerError),
		Entry("handled error", kerrors.NewTimeoutError("timed-out", 200), http.StatusGatewayTimeout),
	)
})
