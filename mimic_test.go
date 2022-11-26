package mimic

import (
	"bytes"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMimic_Close(t *testing.T) {
	tests := []struct {
		name    string
		writer  io.Writer
		wantErr bool
	}{
		{name: "cleanly closes", writer: &bytes.Buffer{}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMimic(WithOutput(tt.writer), WithIdleDuration(10*time.Millisecond))
			assert.NoError(t, err)
			if err := m.Close(); (err != nil) != tt.wantErr {
				t.Errorf("Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMimic_ContainsPattern(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		pattern  []string
		want     bool
	}{
		{name: "simple pattern matches found", contents: "Hello, World!", pattern: []string{
			".*Hello.*",
			".*World.*",
			"[hH]ello,\\s[wW]orld!",
		}, want: true},

		{name: "simple pattern matches not found", contents: "Hello, World!", pattern: []string{
			".*puppies.*",
		}, want: false},

		{name: "complex pattern matches found", contents: "H3ll0,\nWor1d! XXXXXXXXXXXXXXXXXXXXX", pattern: []string{
			"^H3ll0.*",
			"^.*[\n].*[xX]{3,}$",
			".*XXXX$",
			"[h-lH-L0-3]+",
		}, want: true},

		{name: "escapes ansi", contents: "\x1b[38;5;140mfoo\x1b[0m bar", pattern: []string{
			"foo",
			"foo bar",
			"^foo\\sbar$",
			"bar",
		}, want: true},

		{name: "escapes ansi complex", contents: "\x1b[0m\x1b[4m\x1b[42m\x1b[31mfoo\x1b[39m\x1b[49m\x1b[24mfoo\x1b[0m", pattern: []string{
			"^foofoo$",
		}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMimic(WithIdleDuration(15 * time.Millisecond))
			assert.NoError(t, err)

			_, err = m.WriteString(tt.contents)
			assert.NoError(t, err)

			if m.ContainsPattern(tt.pattern...) && !tt.want {
				t.Errorf("ExpectPattern() did not match expected result %v", tt.want)
			}
		})
	}
}

func TestMimic_ExpectString(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		contains []string
		wantErr  bool
	}{
		{name: "simple contents found", contents: "Hello, World!", contains: []string{
			"Hello",
			"World",
			"llo, Wo",
		}, wantErr: false},
		{name: "simple contents with failed match", contents: "Hello, World!", contains: []string{
			"puppies",
		}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMimic(WithIdleDuration(20*time.Millisecond), WithIdleTimeout(50*time.Millisecond))
			assert.NoError(t, err)

			_, err = m.WriteString(tt.contents)
			assert.NoError(t, err)

			if err := m.ExpectString(tt.contains...); (err != nil) != tt.wantErr {
				t.Errorf("ExpectString() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMimic_ExpectPattern(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		pattern  []string
		wantErr  assert.ErrorAssertionFunc
	}{
		{name: "simple pattern matches found", contents: "Hello, World!", pattern: []string{
			".*Hello.*",
			".*World.*",
			"[hH]ello,\\s[wW]orld!",
		}, wantErr: assert.NoError},

		{name: "simple pattern matches not found", contents: "Hello, World!", pattern: []string{
			".*puppies.*",
		}, wantErr: assert.Error},

		{name: "complex pattern with newline should not match as found", contents: "H3ll0,\nWor1d! XXXXXXXXXXXXXXXXXXXXX", pattern: []string{
			"^H3ll0.*",
			"^.*[\n].*[xX]{3,}$",
			".*XXXX$",
			"[h-lH-L0-3]+",
		}, wantErr: assert.NoError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMimic(WithIdleDuration(20*time.Millisecond), WithIdleTimeout(50*time.Millisecond))
			assert.NoError(t, err)

			_, err = m.WriteString(tt.contents)
			assert.NoError(t, err)

			tt.wantErr(t, m.ExpectPattern(tt.pattern...), fmt.Sprintf("ExpectPattern(%v)", tt.pattern))
		})
	}
}
