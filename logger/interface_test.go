package logger

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevelMarshalText(t *testing.T) {
	cases := []struct {
		level LogLevelEnum
		text  string
	}{
		{FATAL, "FATAL"},
		{ERROR, "ERROR"},
		{WARN, "WARN"},
		{LOG, "LOG"},
		{DEBUG, "DEBUG"},
	}
	for _, c := range cases {
		b, err := c.level.MarshalText()
		assert.NoError(t, err)
		assert.Equal(t, c.text, string(b))
	}
}

func TestLevelUnmarshalText(t *testing.T) {
	cases := []struct {
		text  string
		level LogLevelEnum
	}{
		{"FATAL", FATAL},
		{"ERROR", ERROR},
		{"ERR", ERROR},
		{"warn", WARN},
		{"warning", WARN},
		{"INFO", LOG},
		{"LOG", LOG},
		{"DEBUG", DEBUG},
		{"  debug  ", DEBUG},
	}
	for _, c := range cases {
		var l LogLevelEnum
		err := l.UnmarshalText([]byte(c.text))
		assert.NoError(t, err, c.text)
		assert.Equal(t, c.level, l, c.text)
	}

	var l LogLevelEnum
	assert.Error(t, l.UnmarshalText([]byte("nonsense")))
}

func TestLevelJSONRoundtrip(t *testing.T) {
	type cfg struct {
		L LogLevelEnum `json:"l"`
	}
	in := cfg{L: WARN}
	b, err := json.Marshal(in)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"l":"WARN"}`, string(b))

	var out cfg
	assert.NoError(t, json.Unmarshal([]byte(`{"l":"DEBUG"}`), &out))
	assert.Equal(t, DEBUG, out.L)

	// numeric fallback
	var out2 cfg
	assert.NoError(t, json.Unmarshal([]byte(`{"l":3}`), &out2))
	assert.Equal(t, DEBUG, out2.L)

	// null leaves zero value, no error (config UX: optional level field)
	var out3 cfg
	out3.L = WARN
	assert.NoError(t, json.Unmarshal([]byte(`{"l":null}`), &out3))
	assert.Equal(t, WARN, out3.L) // unchanged
}
