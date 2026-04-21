// Copyright 2026 The Joe-cli Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpserver

import (
	"context"
	"net/http"
	"sync"
)

//counterfeiter:generate . handler

type handler = http.Handler

type reloadableMux interface {
	ReloadAll()
}

// ReloadableHandler provides a handler whose internal state can be invalidated
// so that the next time it serves, it recomputes it
type ReloadableHandler interface {
	http.Handler

	Invalidate()
}

// NewReloadableHandler provides a handler which is reloadable. When the server
// starts or when the server is restarted, the function is invoked
// to generate the actual handler that is called. If an
// error is returned, the server will respond with 503 Service Unavailable.
func NewReloadableHandler(fn func(context.Context) (http.Handler, error)) ReloadableHandler {
	return &reloadableHandler{fn: fn}
}

type reloadSupport struct {
	*http.ServeMux

	targets []*reloadableHandler
}

func (r *reloadSupport) ReloadAll() {
	for _, t := range r.targets {
		t.Invalidate()
	}
}

func (r *reloadSupport) Handle(pattern string, handler http.Handler) {
	r.ServeMux.Handle(pattern, handler)

	if reloadable, ok := handler.(*reloadableHandler); ok {
		r.targets = append(r.targets, reloadable)
	}
}

type reloadableHandler struct {
	fn        func(context.Context) (http.Handler, error)
	mu        sync.RWMutex
	handler   http.Handler
	once      sync.Once
	lastError error
}

func (h *reloadableHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler := h.ensureHandler(r.Context())
	handler.ServeHTTP(w, r)
}

func (h *reloadableHandler) ensureHandler(ctx context.Context) http.Handler {
	h.mu.RLock()

	if h.handler != nil {
		h.mu.RUnlock()
		return h.handler
	}
	h.mu.RUnlock()

	h.mu.Lock()
	defer h.mu.Unlock()

	h.once.Do(func() {
		handler, err := h.fn(ctx)
		if err != nil {
			h.lastError = err
			h.handler = http.HandlerFunc(serviceUnavailable)

		} else {
			h.handler = handler
		}
	})

	// If handler creation failed, use the 503 handler
	if h.handler == nil {
		h.handler = http.HandlerFunc(serviceUnavailable)
	}
	return h.handler
}

func (h *reloadableHandler) Invalidate() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.handler = nil
	h.once = sync.Once{}
}

func serviceUnavailable(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
}
