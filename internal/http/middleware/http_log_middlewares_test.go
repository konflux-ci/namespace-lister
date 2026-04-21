package middleware_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"github.com/konflux-ci/namespace-lister/internal/http/middleware"
	"github.com/konflux-ci/namespace-lister/internal/log"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HttpLogMiddlewares", func() {
	newBufferLogger := func(buf *bytes.Buffer) *slog.Logger {
		return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	Describe("AddInjectLoggerMiddleware", func() {
		It("runs the next handler with the injected logger in context", func(ctx context.Context) {
			By("building a JSON logger that writes to a buffer")
			var buf bytes.Buffer
			l := newBufferLogger(&buf)

			By("wiring a next handler that logs using the context logger")
			nhInvoked := false
			nh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nhInvoked = true
				log.GetLoggerFromContext(r.Context()).Info("downstream", "k", "v")
			})

			By("composing and invoking the middleware")
			m := middleware.AddInjectLoggerMiddleware(*l, nh)
			w := httptest.NewRecorder()
			r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
			Expect(err).NotTo(HaveOccurred())
			m.ServeHTTP(w, r)

			By("asserting the next handler ran and logs went to the buffer")
			Expect(nhInvoked).To(BeTrue())
			Expect(buf.String()).To(ContainSubstring(`"msg":"downstream"`))
			Expect(buf.String()).To(ContainSubstring(`"k":"v"`))
		})
	})

	Describe("AddLogCorrelationIDMiddleware", func() {
		It("propagates X-Correlation-ID into the context logger", func(ctx context.Context) {
			By("building a buffer-backed logger and wrapping correlation middleware after inject")
			var buf bytes.Buffer
			l := newBufferLogger(&buf)
			nh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				log.GetLoggerFromContext(r.Context()).Info("during-next")
			})
			h := middleware.AddInjectLoggerMiddleware(*l, middleware.AddLogCorrelationIDMiddleware(nh))

			By("sending a request with a fixed X-Correlation-ID header")
			w := httptest.NewRecorder()
			r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/x", nil)
			Expect(err).NotTo(HaveOccurred())
			r.Header.Set("X-Correlation-ID", "test-cid-1")
			h.ServeHTTP(w, r)

			By("checking logs include the header value")
			Expect(buf.String()).To(ContainSubstring(`"correlation-id":"test-cid-1"`))
			Expect(buf.String()).To(ContainSubstring(`"msg":"during-next"`))
		})

		It("generates a correlation id when the header is absent", func(ctx context.Context) {
			By("building inject + correlation without setting X-Correlation-ID")
			var buf bytes.Buffer
			l := newBufferLogger(&buf)
			nh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				log.GetLoggerFromContext(r.Context()).Info("during-next")
			})
			h := middleware.AddInjectLoggerMiddleware(*l, middleware.AddLogCorrelationIDMiddleware(nh))

			By("invoking the handler chain")
			w := httptest.NewRecorder()
			r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/y", nil)
			Expect(err).NotTo(HaveOccurred())
			h.ServeHTTP(w, r)

			By("expecting a UUID-shaped correlation-id in the JSON log line")
			Expect(buf.String()).To(MatchRegexp(`"correlation-id":"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"`))
		})
	})

	Describe("AddLogRequestMiddleware", func() {
		It("logs the path and completion after the next handler", func(ctx context.Context) {
			By("composing inject then request-log middleware over a no-op next handler")
			var buf bytes.Buffer
			l := newBufferLogger(&buf)
			nh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			h := middleware.AddInjectLoggerMiddleware(*l, middleware.AddLogRequestMiddleware(nh))

			By("serving a GET with a non-trivial path")
			w := httptest.NewRecorder()
			r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/api/foo", nil)
			Expect(err).NotTo(HaveOccurred())
			h.ServeHTTP(w, r)

			By("asserting the post-request log line and request attribute")
			Expect(buf.String()).To(ContainSubstring(`"msg":"request processed"`))
			Expect(buf.String()).To(ContainSubstring(`"request":"/api/foo"`))
		})
	})

	Describe("Chained Log Middlewares", func() {
		It("includes correlation id and request path when chained after inject and correlation", func(ctx context.Context) {
			By("stacking inject, correlation, then request middleware")
			var buf bytes.Buffer
			l := newBufferLogger(&buf)
			nh := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
			h := middleware.AddInjectLoggerMiddleware(*l,
				middleware.AddLogCorrelationIDMiddleware(
					middleware.AddLogRequestMiddleware(nh)))

			By("sending a request with a known correlation id and path")
			w := httptest.NewRecorder()
			r, err := http.NewRequestWithContext(ctx, http.MethodGet, "/z", nil)
			Expect(err).NotTo(HaveOccurred())
			r.Header.Set("X-Correlation-ID", "chain-cid")
			h.ServeHTTP(w, r)

			By("checking the combined log output")
			s := buf.String()
			Expect(s).To(ContainSubstring(`"correlation-id":"chain-cid"`))
			Expect(s).To(ContainSubstring(`"request":"/z"`))
			Expect(s).To(ContainSubstring(`"msg":"request processed"`))
		})
	})
})
