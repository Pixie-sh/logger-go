package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlogHandlerBasic(t *testing.T) {
	var buf bytes.Buffer
	l, err := NewLogger(&buf, "App", "Scope", "uid", DEBUG, []string{"trace_id"})
	assert.NoError(t, err)

	sl := slog.New(NewSlogHandler(l))
	sl.Info("hello", "user", "alice", "n", 7)

	var entry map[string]any
	assert.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry))
	assert.Equal(t, "hello", entry["message"])
	assert.Equal(t, "INFO", entry["level"])
	assert.Equal(t, "alice", entry["user"])
	assert.EqualValues(t, 7, entry["n"])
}

func TestSlogHandlerLevelGating(t *testing.T) {
	var buf bytes.Buffer
	l, err := NewLogger(&buf, "App", "Scope", "uid", WARN, []string{})
	assert.NoError(t, err)

	sl := slog.New(NewSlogHandler(l))
	sl.Debug("invisible")
	sl.Info("invisible")
	sl.Warn("visible-warn")
	sl.Error("visible-err")

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	assert.Len(t, lines, 2)
}

func TestSlogHandlerGroupsAndAttrs(t *testing.T) {
	var buf bytes.Buffer
	l, err := NewLogger(&buf, "App", "Scope", "uid", DEBUG, []string{})
	assert.NoError(t, err)

	sl := slog.New(NewSlogHandler(l)).With("app_id", 1).WithGroup("req")
	sl.Info("hit", "path", "/x")

	var entry map[string]any
	assert.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry))
	assert.EqualValues(t, 1, entry["app_id"])
	assert.Equal(t, "/x", entry["req.path"])
}

func TestSlogHandlerCtxPassthrough(t *testing.T) {
	var buf bytes.Buffer
	l, err := NewLogger(&buf, "App", "Scope", "uid", DEBUG, []string{TraceID})
	assert.NoError(t, err)

	sl := slog.New(NewSlogHandler(l))
	ctx := context.WithValue(context.Background(), TraceID, "abc-123")
	sl.InfoContext(ctx, "hi")

	var entry map[string]any
	assert.NoError(t, json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &entry))
	ctxMap, _ := entry["ctx"].(map[string]any)
	assert.Equal(t, "abc-123", ctxMap[TraceID])
}
