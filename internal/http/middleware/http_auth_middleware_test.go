package middleware_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"

	nscontext "github.com/konflux-ci/namespace-lister/internal/context"
	"github.com/konflux-ci/namespace-lister/internal/http/middleware"
	"github.com/konflux-ci/namespace-lister/internal/http/middleware/mocks"
	"github.com/konflux-ci/namespace-lister/internal/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
)

var _ = Describe("HttpAuthMiddleware", func() {
	newBufferLogger := func(buf *bytes.Buffer) *slog.Logger {
		return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	var authenticatorRequest *mocks.MockRequest

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())

		authenticatorRequest = mocks.NewMockRequest(ctrl)
	})

	It("returns Unauthorized on denied requests", func(ctx context.Context) {
		By("building the request the middleware will pass to the authenticator")
		r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		Expect(err).NotTo(HaveOccurred())

		By("set up an always-deny authenticator")
		authenticatorRequest.EXPECT().AuthenticateRequest(r).
			Times(1).
			Return(nil, false, nil)
		m := middleware.AddAuthnMiddleware(authenticatorRequest, nil)

		By("execute the middleware")
		w := httptest.NewRecorder()
		m.ServeHTTP(w, r)

		By("check the StatusCode is Unauthorized")
		Expect(w.Result().StatusCode).To(Equal(http.StatusUnauthorized))
	})

	It("returns Unauthorized on errored requests", func(ctx context.Context) {
		By("building a JSON logger that writes to a buffer")
		var buf bytes.Buffer
		l := newBufferLogger(&buf)

		By("building the request with a context logger and a header for the error log")
		r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		Expect(err).NotTo(HaveOccurred())
		r.Header.Set("X-Test-Auth", "probe")
		r = r.WithContext(log.SetLoggerIntoContext(r.Context(), l))

		By("set up an always-erroring authenticator")
		authErr := errors.New("error contacting the APIServer")
		authenticatorRequest.EXPECT().AuthenticateRequest(r).
			Times(1).
			Return(nil, false, authErr)
		m := middleware.AddAuthnMiddleware(authenticatorRequest, nil)

		By("execute the middleware")
		w := httptest.NewRecorder()
		m.ServeHTTP(w, r)

		By("check the StatusCode is Unauthorized")
		Expect(w.Result().StatusCode).To(Equal(http.StatusUnauthorized))

		By("check the error log includes the failure, headers, and message")
		logged := buf.String()
		Expect(logged).To(ContainSubstring(`"msg":"error authenticating request"`))
		Expect(logged).To(ContainSubstring(`"error contacting the APIServer"`))
		Expect(logged).To(ContainSubstring(`"X-Test-Auth"`))
		Expect(logged).To(ContainSubstring(`"probe"`))
	})

	It("runs the next handler if authentication pass", func(ctx context.Context) {
		By("building the request the middleware will pass to the authenticator")
		r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		Expect(err).NotTo(HaveOccurred())

		By("set up an always-passing authenticator")
		rs := &authenticator.Response{User: &user.DefaultInfo{}}
		authenticatorRequest.EXPECT().AuthenticateRequest(r).
			Times(1).
			Return(rs, true, nil)

		By("set up an validating next HTTP handler")
		nhInvoked := false
		nh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nhInvoked = true
			Expect(r).NotTo(BeNil())
			Expect(r.Context()).NotTo(BeNil())
			Expect(r.Context().Value(nscontext.ContextKeyUserDetails)).To(Equal(rs))
		})

		By("set up the Authn Middleware")
		m := middleware.AddAuthnMiddleware(authenticatorRequest, nh)

		By("execute the middleware")
		w := httptest.NewRecorder()
		m.ServeHTTP(w, r)

		By("check the next handler has been invoked")
		Expect(nhInvoked).To(BeTrue())
	})
})
