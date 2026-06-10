package logger

import (
	"context"
	"log/slog"
	"time"

	"github.com/pixie-sh/logger-go/caller"
)

// SlogHandler is an slog.Handler that delegates to an Interface so callers
// can plug this library into the standard log/slog ecosystem:
//
//	h := logger.NewSlogHandler(logger.Logger)
//	sl := slog.New(h)
//	sl.Info("hello", "user", "alice")
//
// Behavior:
//   - slog.LogValuer values are Resolve()'d before being attached, so types
//     that implement redaction or lazy evaluation work as advertised.
//   - slog.Group attributes are flattened into dot-separated keys
//     ("http.status", etc.), matching the WithGroup convention.
//   - r.PC and r.Time are honored when the underlying logger is one of this
//     package's concrete types — the caller and timestamp reflect the
//     original sl.Info(...) site rather than the handler's frame.
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
	return slogLevelEnabled(h.l.Level(), level)
}

// Handle converts an slog.Record into a call against the underlying logger.
func (h *SlogHandler) Handle(ctx context.Context, r slog.Record) error {
	l := h.l.Clone()
	if ctx != nil {
		l = l.WithCtx(ctx)
	}
	r.Attrs(func(a slog.Attr) bool {
		l = applyAttr(l, h.group, a)
		return true
	})

	level := slogToInternal(r.Level)

	// When the underlying logger is one of our concrete types we have an
	// internal path that takes the pre-resolved caller and timestamp.
	// Otherwise fall back to the Interface method and accept the depth/time
	// drift for that adapter.
	if e, ok := l.(internalEmit); ok {
		var c caller.Ptr
		if r.PC != 0 {
			c = caller.FromPC(r.PC)
		}
		t := r.Time
		if t.IsZero() {
			t = time.Now()
		}
		e.emitAt(level, c, t, "%s", r.Message)
		return nil
	}

	switch level {
	case ERROR:
		l.Error("%s", r.Message)
	case WARN:
		l.Warn("%s", r.Message)
	case LOG:
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
		c = applyAttr(c, h.group, a)
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

func qualify(group, key string) string {
	if group == "" {
		return key
	}
	return group + "." + key
}

// applyAttr attaches one slog.Attr to the logger, resolving LogValuer
// values and recursively flattening Group attrs into dotted keys.
// Returns the (possibly new) logger to thread through chains.
func applyAttr(l Interface, group string, a slog.Attr) Interface {
	// slog spec: empty attrs are ignored.
	if a.Equal(slog.Attr{}) {
		return l
	}
	// Resolve LogValuer chains so redacting / lazy values see the right
	// substitute before extraction.
	v := a.Value.Resolve()

	if v.Kind() == slog.KindGroup {
		// Inline group attrs get flattened under the parent key (or, if
		// the group name is empty, inlined at the current level).
		childGroup := group
		if a.Key != "" {
			childGroup = qualify(group, a.Key)
		}
		for _, child := range v.Group() {
			l = applyAttr(l, childGroup, child)
		}
		return l
	}

	return l.With(qualify(group, a.Key), v.Any())
}

func slogToInternal(level slog.Level) LogLevelEnum {
	switch {
	case level >= slog.LevelError:
		return ERROR
	case level >= slog.LevelWarn:
		return WARN
	case level >= slog.LevelInfo:
		return LOG
	default:
		return DEBUG
	}
}

func slogLevelEnabled(myLevel LogLevelEnum, level slog.Level) bool {
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
