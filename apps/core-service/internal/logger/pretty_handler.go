package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"sync"
)

type prettyHandler struct {
	w    io.Writer
	opts *slog.HandlerOptions
	mu   *sync.Mutex
	ops  []func(slog.Handler) slog.Handler
}

func newPrettyHandler(w io.Writer, opts *slog.HandlerOptions) *prettyHandler {
	return &prettyHandler{w: w, opts: opts, mu: &sync.Mutex{}}
}

func (h *prettyHandler) build(w io.Writer) slog.Handler {
	var handler slog.Handler = slog.NewJSONHandler(w, h.opts)
	for _, op := range h.ops {
		handler = op(handler)
	}
	return handler
}

func (h *prettyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.build(io.Discard).Enabled(ctx, level)
}

func (h *prettyHandler) Handle(ctx context.Context, record slog.Record) error {
	var compact bytes.Buffer
	if err := h.build(&compact).Handle(ctx, record); err != nil {
		return err
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, bytes.TrimRight(compact.Bytes(), "\n"), "", "  "); err != nil {
		h.mu.Lock()
		defer h.mu.Unlock()
		_, writeErr := h.w.Write(compact.Bytes())
		return writeErr
	}
	pretty.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(pretty.Bytes())
	return err
}

func (h *prettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.clone(func(handler slog.Handler) slog.Handler {
		return handler.WithAttrs(attrs)
	})
}

func (h *prettyHandler) WithGroup(name string) slog.Handler {
	return h.clone(func(handler slog.Handler) slog.Handler {
		return handler.WithGroup(name)
	})
}

func (h *prettyHandler) clone(op func(slog.Handler) slog.Handler) *prettyHandler {
	ops := make([]func(slog.Handler) slog.Handler, len(h.ops)+1)
	copy(ops, h.ops)
	ops[len(h.ops)] = op

	return &prettyHandler{w: h.w, opts: h.opts, mu: h.mu, ops: ops}
}
