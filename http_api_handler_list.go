package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/konflux-ci/namespace-lister/internal/constant"
	nscontext "github.com/konflux-ci/namespace-lister/internal/context"
	"github.com/konflux-ci/namespace-lister/internal/log"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/authentication/authenticator"
)

var _ http.Handler = &ListNamespacesHandler{}

type ListNamespacesHandler struct {
	lister NamespaceLister
}

func NewListNamespacesHandler(lister NamespaceLister) http.Handler {
	return &ListNamespacesHandler{
		lister: lister,
	}
}

func (h *ListNamespacesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := log.GetLoggerFromContext(ctx)

	ud := r.Context().Value(nscontext.ContextKeyUserDetails).(*authenticator.Response)

	// retrieve projects as the user
	nn, err := h.lister.ListNamespaces(r.Context(), ud.User.GetName(), ud.User.GetGroups())
	if err != nil {
		serr := &kerrors.StatusError{}
		if errors.As(err, &serr) {
			http.Error(w, serr.Error(), int(serr.Status().Code))
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// build response
	// for PoC limited to JSON
	b, err := json.Marshal(nn)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Add(constant.HttpContentType, constant.HttpContentTypeApplication)
	h.write(l, w, b)
}

func (h *ListNamespacesHandler) write(l *slog.Logger, w http.ResponseWriter, data []byte) bool {
	if _, err := w.Write(data); err != nil {
		l.Error("error writing reply", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return false
	}
	return true
}
