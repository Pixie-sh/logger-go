package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// --- #2: typed-nil error must not panic the parser ---

type typedNilErr struct{ msg string }

func (e *typedNilErr) Error() string {
	if e == nil {
		// In real-world code this would commonly panic on nil receiver.
		// The parser must not reach this path.
		panic("Error() called on nil receiver")
	}
	return e.msg
}

func TestParserSurvivesTypedNilError(t *testing.T) {
	var buf bytes.Buffer
	l, err := NewLogger(&buf, "App", "Scope", "uid", DEBUG, nil)
	assert.NoError(t, err)

	var nilErr *typedNilErr // typed-nil
	assert.NotPanics(t, func() {
		l.With("err", nilErr).Error("typed-nil err")
	}, "DefaultJSONParser must treat typed-nil error as nil and not call .Error()")

	var entry map[string]any
	assert.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry))
	assert.Equal(t, "nil", entry["err"], "typed-nil should marshal as the string 'nil'")
}

func TestTextParserSurvivesTypedNilError(t *testing.T) {
	var buf bytes.Buffer
	l, err := NewLogger(&buf, "App", "Scope", "uid", DEBUG, nil, DefaultTextParser)
	assert.NoError(t, err)
	var nilErr *typedNilErr
	assert.NotPanics(t, func() { l.With("err", nilErr).Error("x") })
	assert.Contains(t, buf.String(), "Fields.err: nil")
}

// --- #3: flatten must not blow the stack on a self-referential struct ---

type cycle struct {
	Name string
	Self *cycle
}

func TestTextParserBoundsRecursionOnCycle(t *testing.T) {
	var buf bytes.Buffer
	l, err := NewLogger(&buf, "App", "Scope", "uid", DEBUG, nil, DefaultTextParser)
	assert.NoError(t, err)
	c := &cycle{Name: "root"}
	c.Self = c
	assert.NotPanics(t, func() { l.With("c", c).Error("cycle") })
	assert.Contains(t, buf.String(), "max depth reached")
}

// --- #4: Fatal must call os.Exit even if the parser panics ---
// We can't actually call os.Exit in a test. Instead, verify that the Fatal
// method uses the defer-os.Exit pattern by stubbing the parser to panic and
// confirming Fatal does NOT return (it would, with the bug, since the panic
// would propagate around os.Exit). We do this in a subprocess.

func TestFatalDefersExitAcrossParserPanic(t *testing.T) {
	// Swap exitFn with a stub so we can observe whether the defer fired.
	orig := exitFn
	defer func() { exitFn = orig }()
	var exitCode int
	exitCalled := false
	exitFn = func(code int) {
		exitCode = code
		exitCalled = true
		// Production os.Exit kills the process; in tests we have to swallow
		// the in-flight panic ourselves so the test goroutine returns. The
		// `recover()` runs because exitFn is invoked from a defer.
		recover()
	}

	var buf bytes.Buffer
	parserPanicked := false
	panicParser := ParserFn(func(e *Entry) []byte {
		parserPanicked = true
		panic("parser boom")
	})
	l, err := NewLogger(&buf, "App", "Scope", "uid", DEBUG, nil, panicParser)
	assert.NoError(t, err)

	// Fatal's `defer exitFn(1)` must run on the parser panic, so the panic
	// is consumed by the deferred call (which now just records the code).
	// Without the defer, the panic would unwind past Fatal — we'd have to
	// recover here to keep the test alive.
	defer func() {
		r := recover()
		assert.Nil(t, r, "defer exitFn should run before the panic escapes Fatal")
		assert.True(t, exitCalled, "Fatal must defer-call exitFn even when emit panics")
		assert.Equal(t, 1, exitCode)
		assert.True(t, parserPanicked, "parser must have been invoked")
	}()
	l.Fatal("boom")
}

// --- #5: SetLevel on a Clone must not affect the original ---

func TestCloneIsolatesSetLevel(t *testing.T) {
	parent, err := NewLogger(nil, "App", "Scope", "uid", DEBUG, nil)
	assert.NoError(t, err)
	child := parent.Clone()
	child.SetLevel(ERROR)
	assert.Equal(t, DEBUG, parent.Level(), "*logger.Clone: parent level must be unchanged")
	assert.Equal(t, ERROR, child.Level())

	inner := parent.With("k", "v")
	innerClone := inner.Clone()
	innerClone.SetLevel(WARN)
	assert.Equal(t, DEBUG, inner.Level(), "innerLogger.Clone: parent level must be unchanged")
	assert.Equal(t, WARN, innerClone.Level())
}

// --- #10: factory must not alias caller's ExpectedCtxFields slice ---

func TestFactoryDoesNotAliasExpectedCtxFields(t *testing.T) {
	original := make([]string, 1, 4) // cap > len: this is the footgun
	original[0] = "request_id"

	factory, _ := NewFactory(DefaultFactoryConfiguration)
	_, err := factory.Create(Configuration{
		Driver:            JSONLoggerDriver,
		LogLevel:          LOG,
		ExpectedCtxFields: original,
		Writer:            &bytes.Buffer{},
	})
	assert.NoError(t, err)

	// The slot beyond len(original) must NOT have been overwritten with
	// "trace_id" by the factory's internal append.
	tail := original[:cap(original)]
	for i := len(original); i < cap(original); i++ {
		assert.Empty(t, tail[i], "factory leaked into caller's slice backing array at index %d", i)
	}
}

// --- #11: ctx key present even when no expected fields match ---

func TestCtxKeyAlwaysPresentWithWithCtx(t *testing.T) {
	var buf bytes.Buffer
	l, err := NewLogger(&buf, "App", "Scope", "uid", DEBUG, []string{"missing_key"})
	assert.NoError(t, err)
	ctx := context.WithValue(context.Background(), "unrelated", "x")
	l.WithCtx(ctx).Log("hi")

	var entry map[string]any
	assert.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry))
	_, has := entry["ctx"]
	assert.True(t, has, "ctx key must remain in the JSON schema once WithCtx has been called")
}

// --- #12: UnmarshalJSON of null leaves zero value ---
// covered in interface_test.go TestLevelJSONRoundtrip

// --- #13: must() catches typed-nil interface assignment ---

type nilLoggerImpl struct{}

func (*nilLoggerImpl) Clone() Interface                       { return nil }
func (*nilLoggerImpl) WithCtx(ctx context.Context) Interface  { return nil }
func (*nilLoggerImpl) With(field string, value any) Interface { return nil }
func (*nilLoggerImpl) Level() LogLevelEnum                    { return LOG }
func (*nilLoggerImpl) SetLevel(LogLevelEnum)                  {}
func (*nilLoggerImpl) Log(string, ...any)                     {}
func (*nilLoggerImpl) Info(string, ...any)                    {}
func (*nilLoggerImpl) Error(string, ...any)                   {}
func (*nilLoggerImpl) Warn(string, ...any)                    {}
func (*nilLoggerImpl) Debug(string, ...any)                   {}
func (*nilLoggerImpl) Fatal(string, ...any)                   {}

func TestMustCatchesTypedNil(t *testing.T) {
	orig := Logger
	defer func() { Logger = orig }()

	var typed *nilLoggerImpl // typed nil that satisfies Interface
	SetLogger(typed)
	assert.Panics(t, func() { Log("x") }, "must() should panic on typed-nil Interface")
}

// --- #15: SetLogger is race-safe ---

func TestSetLoggerRaceSafe(t *testing.T) {
	orig := Logger
	defer func() { Logger = orig }()

	l1, _ := NewLogger(&bytes.Buffer{}, "A", "S", "u", LOG, nil)
	l2, _ := NewLogger(&bytes.Buffer{}, "B", "S", "u", LOG, nil)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				SetLogger(l1)
			} else {
				SetLogger(l2)
			}
		}(i)
		go func() {
			defer wg.Done()
			Log("concurrent")
		}()
	}
	wg.Wait()
}

// --- #6, #7, #1: slog handler honors r.PC, r.Time, Resolves LogValuer, flattens Group ---

type redactedPassword string

func (redactedPassword) LogValue() slog.Value { return slog.StringValue("REDACTED") }

func TestSlogHandlerResolvesLogValuer(t *testing.T) {
	var buf bytes.Buffer
	l, _ := NewLogger(&buf, "App", "Scope", "uid", DEBUG, nil)
	sl := slog.New(NewSlogHandler(l))
	sl.Info("login", "pw", redactedPassword("hunter2"))

	var entry map[string]any
	assert.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry))
	assert.Equal(t, "REDACTED", entry["pw"], "LogValuer must be Resolve()'d before storage")
	assert.NotContains(t, buf.String(), "hunter2", "raw secret must not appear in output")
}

func TestSlogHandlerFlattensInlineGroup(t *testing.T) {
	var buf bytes.Buffer
	l, _ := NewLogger(&buf, "App", "Scope", "uid", DEBUG, nil)
	sl := slog.New(NewSlogHandler(l))
	sl.Info("req",
		slog.Group("http",
			slog.String("method", "GET"),
			slog.Int("status", 200),
		),
	)
	var entry map[string]any
	assert.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry))
	assert.Equal(t, "GET", entry["http.method"])
	assert.EqualValues(t, 200, entry["http.status"])
}

func TestSlogHandlerHonorsRecordTimeAndPC(t *testing.T) {
	var buf bytes.Buffer
	l, _ := NewLogger(&buf, "App", "Scope", "uid", DEBUG, nil)
	sl := slog.New(NewSlogHandler(l))

	// Build a record at a known PC/time and submit it directly so we control
	// both. The PC is from this test function — Handle should report it as
	// the caller, NOT some frame inside slog_handler.go.
	pcs := make([]uintptr, 1)
	runtime.Callers(1, pcs)
	knownTime := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)

	rec := slog.NewRecord(knownTime, slog.LevelInfo, "hi", pcs[0])
	err := sl.Handler().Handle(context.Background(), rec)
	assert.NoError(t, err)

	var entry map[string]any
	assert.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry))

	c, ok := entry["caller"].(map[string]any)
	assert.True(t, ok, "caller field present")
	path, _ := c["Path"].(string)
	assert.True(t, strings.Contains(path, "TestSlogHandlerHonorsRecordTimeAndPC"),
		"caller path %q should resolve to the test function via r.PC, not the handler", path)

	ts, _ := entry["timestamp"].(string)
	assert.True(t, strings.HasPrefix(ts, "2020-01-02"), "timestamp should mirror r.Time, got %q", ts)
}

// --- #8: package-level helpers (logger.Log etc.) report user as caller ---

func TestSingletonHelpersReportUserAsCaller(t *testing.T) {
	orig := Logger
	defer func() { Logger = orig }()

	var buf bytes.Buffer
	l, _ := NewLogger(&buf, "App", "Scope", "uid", DEBUG, nil)
	SetLogger(l)

	Log("hi") // this line is the expected caller

	var entry map[string]any
	assert.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry))
	c, _ := entry["caller"].(map[string]any)
	path, _ := c["Path"].(string)
	assert.Contains(t, path, "TestSingletonHelpersReportUserAsCaller",
		"package-level Log() must attribute the user's frame, not the singleton wrapper")
}

// --- #14: write path must not corrupt a pooled-buffer-backed parser ---

func TestWritePathDoesNotAppendIntoCallerSlice(t *testing.T) {
	// Custom parser that returns a slice with cap > len — exactly the shape
	// a pool-backed buffer would return.
	var buf bytes.Buffer
	parser := ParserFn(func(e *Entry) []byte {
		out := make([]byte, 1, 64) // len=1, cap=64
		out[0] = 'x'
		return out
	})
	l, err := NewLogger(&buf, "App", "Scope", "uid", DEBUG, nil, parser)
	assert.NoError(t, err)
	l.Log("hi")
	assert.Equal(t, "x\n", buf.String(), "writer received payload + newline as two writes")
}

// --- generic: errors.Is unwrap chain still surfaces ---

func TestErrorUnwrapStillSurfaced(t *testing.T) {
	var buf bytes.Buffer
	l, _ := NewLogger(&buf, "App", "Scope", "uid", DEBUG, nil)
	root := errors.New("inner")
	wrapped := errFmtWrap{msg: "outer", cause: root}
	l.With("err", wrapped).Error("boom")

	var entry map[string]any
	assert.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry))
	ei, _ := entry["err"].(map[string]any)
	assert.Equal(t, "outer", ei["error"])
	assert.Equal(t, "inner", ei["error.unwrap"])
}

type errFmtWrap struct {
	msg   string
	cause error
}

func (e errFmtWrap) Error() string { return e.msg }
func (e errFmtWrap) Unwrap() error { return e.cause }
