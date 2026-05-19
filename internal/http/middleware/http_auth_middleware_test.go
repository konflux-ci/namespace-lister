package middleware_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"github.com/konflux-ci/namespace-lister/internal/contextkey"
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
		opts := &slog.HandlerOptions{Level: slog.Level(-1000)} // log at maximum verbosity
		return slog.New(slog.NewJSONHandler(buf, opts))
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
		headers := map[string][]string{
			"X-Test-Auth": []string{"probe"},
		}

		By("building a JSON logger that writes to a buffer")
		var buf bytes.Buffer
		l := newBufferLogger(&buf)

		By("building the request with a context logger and a header for the error log")
		r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		Expect(err).NotTo(HaveOccurred())
		for k, vv := range headers {
			for _, v := range vv {
				r.Header.Set(k, v)
			}
		}
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

		By("check the error log includes the failure and message")
		logged := buf.String()
		Expect(logged).To(And(
			ContainSubstring(`"msg":"error authenticating request"`),
			ContainSubstring(`"error contacting the APIServer"`),
		))

		By("check that the error log doesn't include headers")
		for k, hv := range headers {
			Expect(logged).To(Not(ContainSubstring(k)))
			for _, h := range hv {
				Expect(logged).NotTo(ContainSubstring(h))
			}
		}
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
			Expect(r.Context().Value(contextkey.ContextKeyUserDetails)).To(Equal(rs))
		})

		By("set up the Authn Middleware")
		m := middleware.AddAuthnMiddleware(authenticatorRequest, nh)

		By("setting up a response recoder")
		w := httptest.NewRecorder()
		w.Code = 0

		By("execute the middleware")
		m.ServeHTTP(w, r)

		By("check the next handler has been invoked")
		Expect(nhInvoked).To(BeTrue())

		By("checking status was not written to the response")
		Expect(w.Code).To(BeZero())
	})

	It("should log the authenticated request at debug level", func(ctx context.Context) {
		By("building a JSON logger")
		var buf bytes.Buffer
		l := newBufferLogger(&buf)

		By("building the request with the debug logger injected")
		r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		Expect(err).NotTo(HaveOccurred())
		r = r.WithContext(log.SetLoggerIntoContext(r.Context(), l))

		By("set up an always-passing authenticator")
		rs := &authenticator.Response{User: &user.DefaultInfo{}}
		authenticatorRequest.EXPECT().AuthenticateRequest(r).
			Times(1).
			Return(rs, true, nil)

		By("set up a no-op next handler")
		nh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

		By("execute the middleware")
		m := middleware.AddAuthnMiddleware(authenticatorRequest, nh)
		w := httptest.NewRecorder()
		m.ServeHTTP(w, r)

		By("check the debug log includes the authentication message and user details")
		logged := buf.String()
		Expect(logged).To(And(
			ContainSubstring("request authenticated"),
			ContainSubstring("user"),
		))
	})

	It("does not panic when authenticator returns nil response with ok", func(ctx context.Context) {
		By("building the request the middleware will pass to the authenticator")
		r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		Expect(err).NotTo(HaveOccurred())

		By("set up an authenticator that returns nil response with ok=true")
		authenticatorRequest.EXPECT().AuthenticateRequest(r).
			Times(1).
			Return(nil, true, nil)

		By("set up a validating next HTTP handler")
		nhInvoked := false
		nh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nhInvoked = true
			Expect(r.Context().Value(contextkey.ContextKeyUserDetails)).To(BeNil())
			w.WriteHeader(http.StatusOK)
		})

		By("set up the Authn Middleware")
		m := middleware.AddAuthnMiddleware(authenticatorRequest, nh)

		By("execute the middleware")
		w := httptest.NewRecorder()
		m.ServeHTTP(w, r)

		By("check the next handler has been invoked")
		Expect(nhInvoked).To(BeTrue())

		By("check the StatusCode is OK")
		Expect(w.Result().StatusCode).To(Equal(http.StatusOK))
	})
})
