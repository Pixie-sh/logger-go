package logger

import (
	"context"
	"log/slog"
)

// SlogHandler is an slog.Handler that delegates to an Interface so callers
// can plug this library into the standard log/slog ecosystem:
//
//	h := logger.NewSlogHandler(logger.Logger)
//	sl := slog.New(h)
//	sl.Info("hello", "user", "alice")
//
// Groups are flattened with a "." separator into the field key.
type SlogHandler struct {
	l     Interface
	group string
}

// NewSlogHandler wraps the given Interface as an slog.Handler.
func NewSlogHandler(l Interface) *SlogHandler {
	return &SlogHandler{l: l}
}

// Enabled reports whether the underlying logger would emit at the given level.
func (h *SlogHandler) Enabled(_ context.Context, level slog.Level) bool {
	myLevel := h.l.Level()
	switch {
	case level >= slog.LevelError:
		return myLevel >= ERROR
	case level >= slog.LevelWarn:
		return myLevel >= WARN
	case level >= slog.LevelInfo:
		return myLevel >= LOG
	default:
		return myLevel >= DEBUG
	}
}

// Handle converts an slog.Record into a call against the underlying logger.
func (h *SlogHandler) Handle(ctx context.Context, r slog.Record) error {
	l := h.l.Clone()
	if ctx != nil {
		l = l.WithCtx(ctx)
	}
	r.Attrs(func(a slog.Attr) bool {
		l = l.With(h.qualify(a.Key), a.Value.Any())
		return true
	})
	// Pass the message as a literal format string to skip fmt parsing.
	switch {
	case r.Level >= slog.LevelError:
		l.Error("%s", r.Message)
	case r.Level >= slog.LevelWarn:
		l.Warn("%s", r.Message)
	case r.Level >= slog.LevelInfo:
		l.Info("%s", r.Message)
	default:
		l.Debug("%s", r.Message)
	}
	return nil
}

// WithAttrs returns a handler that attaches the given attrs to every record.
func (h *SlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	c := h.l.Clone()
	for _, a := range attrs {
		c = c.With(h.qualify(a.Key), a.Value.Any())
	}
	return &SlogHandler{l: c, group: h.group}
}

// WithGroup returns a handler that prefixes subsequent attr keys with name.
func (h *SlogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	g := name
	if h.group != "" {
		g = h.group + "." + name
	}
	return &SlogHandler{l: h.l, group: g}
}

func (h *SlogHandler) qualify(key string) string {
	if h.group == "" {
		return key
	}
	return h.group + "." + key
}
