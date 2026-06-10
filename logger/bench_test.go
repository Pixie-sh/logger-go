package logger

import (
	"context"
	"io"
	"testing"
)

func BenchmarkJSONLog(b *testing.B) {
	l, err := NewLogger(io.Discard, "App", "Scope", "uid", DEBUG, []string{TraceID})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Log("hello world")
	}
}

func BenchmarkJSONLogWithFields(b *testing.B) {
	l, err := NewLogger(io.Discard, "App", "Scope", "uid", DEBUG, []string{TraceID})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.With("user", "alice").With("n", 42).Log("hello %s", "world")
	}
}

func BenchmarkJSONLogWithCtx(b *testing.B) {
	l, err := NewLogger(io.Discard, "App", "Scope", "uid", DEBUG, []string{TraceID})
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), TraceID, "trace-abc")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.WithCtx(ctx).With("user", "alice").Log("hi")
	}
}

func BenchmarkTextLog(b *testing.B) {
	l, err := NewLogger(io.Discard, "App", "Scope", "uid", DEBUG, []string{TraceID}, DefaultTextParser)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Log("hello world")
	}
}
