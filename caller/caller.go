package caller

import (
	"path"
	"runtime"
	"strings"
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
	Path    string        `json:"Path,omitempty"`
	pc      uintptr       `json:"pc,omitempty"`
	details *runtime.Func `json:"details,omitempty"`
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

// NewCaller returns a caller based on depth
func NewCaller(depth Depth) Ptr {
	caller := Caller{}
	pc, _, _, ok := runtime.Caller(depth)
	details := runtime.FuncForPC(pc)
	if ok && details != nil {
		caller.Path = sanitizeCallerPath(path.Base(details.Name()))
		caller.pc = pc
		caller.details = details
	}

	return &caller
}

func sanitizeCallerPath(path string) string {
	rawParts := strings.Split(path, ".")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		trimmed := strings.Trim(
			part,
			"()*",
		)
		parts = append(parts, trimmed)
	}

	return strings.Join(parts, ".")
}
