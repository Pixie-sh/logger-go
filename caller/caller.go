// Package caller resolves runtime call sites for log records.
package caller

import (
	"path"
	"runtime"
	"strings"
	"sync"
)

// Depth caller depth type
type Depth = int

// most used depth for caller
const (
	SelfCallerDepth      Depth = 1
	FnCallerDepth        Depth = 2
	TwoHopsCallerDepth   Depth = 3
	ThreeHopsCallerDepth Depth = 4
	FourHopsCallerDepth  Depth = 5
)

// Ptr is the caller ptr type
type Ptr = *Caller

// Caller holds the caller info. mostly used for metrics
type Caller struct {
	Path    string  `json:"Path,omitempty"`
	pc      uintptr `json:"-"`
	details *runtime.Func
}

// String return caller Path
func (c Caller) String() string {
	return c.Path
}

// Self to be used when client wants his Ptr
func Self() Ptr {
	return NewCaller(FnCallerDepth)
}

// Upper to be used when a client wants his Caller Ptr
func Upper() Ptr {
	return NewCaller(TwoHopsCallerDepth)
}

// pathCache memoizes the sanitized function path keyed by program counter.
// Hot path: caller.Upper() is invoked on every log line.
var pathCache sync.Map // map[uintptr]string

// NewCaller returns a caller based on depth
func NewCaller(depth Depth) Ptr {
	pc, _, _, ok := runtime.Caller(depth)
	if !ok {
		return &Caller{}
	}
	return FromPC(pc)
}

// FromPC resolves a Caller from an already-known program counter (e.g. a
// slog.Record.PC). The path string is memoized.
func FromPC(pc uintptr) Ptr {
	if pc == 0 {
		return &Caller{}
	}
	if cached, hit := pathCache.Load(pc); hit {
		return &Caller{Path: cached.(string), pc: pc}
	}
	details := runtime.FuncForPC(pc)
	if details == nil {
		return &Caller{pc: pc}
	}
	p := sanitizeCallerPath(path.Base(details.Name()))
	pathCache.Store(pc, p)
	return &Caller{Path: p, pc: pc, details: details}
}

func sanitizeCallerPath(p string) string {
	rawParts := strings.Split(p, ".")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		parts = append(parts, strings.Trim(part, "()*"))
	}
	return strings.Join(parts, ".")
}
