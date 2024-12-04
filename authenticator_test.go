package main_test

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/apiserver/pkg/authentication/authenticator"

	namespacelister "github.com/konflux-ci/namespace-lister"
	"github.com/konflux-ci/namespace-lister/mocks"
)

var _ = Describe("Authenticator", func() {

	var (
		ctrl *gomock.Controller
		auth authenticator.Request
		c    *mocks.MockFakeInterface
	)

	userHeaderKey := "X-User-Header"
	userHeaderValue := "my-user"

	BeforeEach(func(ctx context.Context) {
		ctrl = gomock.NewController(GinkgoT())
	})

	When("Header authentication is enabled", func() {
		BeforeEach(func() {
			// given
			c = mocks.NewMockFakeInterface(ctrl)
			a, err := namespacelister.NewAuthenticator(namespacelister.AuthenticatorOptions{
				Client: c,
				Header: userHeaderKey,
			})
			Expect(err).NotTo(HaveOccurred())

			auth = a
		})

		It("returns user info from header", func(ctx context.Context) {
			// given
			r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			Expect(err).NotTo(HaveOccurred())
			r.Header.Add(userHeaderKey, userHeaderValue)

			// when
			rs, ok, err := auth.AuthenticateRequest(r)

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(rs).NotTo(BeNil())
			Expect(rs.User).NotTo(BeNil())
			Expect(rs.User.GetName()).To(BeEquivalentTo(userHeaderValue))
		})
	})

	When("Header authentication is disabled", func() {
		BeforeEach(func() {
			// given
			c = mocks.NewMockFakeInterface(ctrl)
			a, err := namespacelister.NewAuthenticator(namespacelister.AuthenticatorOptions{
				Client: c,
			})
			Expect(err).NotTo(HaveOccurred())

			auth = a
		})

		It("ignores user info from header", func(ctx context.Context) {
			// given
			r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			Expect(err).NotTo(HaveOccurred())
			r.Header.Add(userHeaderKey, userHeaderValue)

			// when
			rs, ok, err := auth.AuthenticateRequest(r)

			// then
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(rs).To(BeNil())
		})

		It("tries to validate the bearer token", func(ctx context.Context) {
			// we expect the TokenReview API to be used to validate the token
			c.EXPECT().Post().Times(1)

			// given
			r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			Expect(err).NotTo(HaveOccurred())
			r.Header.Add("Authorization", "Bearer invalid")

			// when
			rs, ok, err := auth.AuthenticateRequest(r)

			// then
			Expect(err).To(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(rs).To(BeNil())
		})
	})
})
