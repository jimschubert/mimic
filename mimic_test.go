package mimic

import (
	"bytes"
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
		wantErr  bool
	}{
		{name: "simple pattern matches found", contents: "Hello, World!", pattern: []string{
			".*Hello.*",
			".*World.*",
			"[hH]ello,\\s[wW]orld!",
		}, wantErr: false},

		{name: "simple pattern matches not found", contents: "Hello, World!", pattern: []string{
			".*puppies.*",
		}, wantErr: true},

		{name: "complex pattern matches found", contents: "H3ll0,\nWor1d! XXXXXXXXXXXXXXXXXXXXX", pattern: []string{
			"^H3ll0.*",
			"^.*[\n].*[xX]{3,}$",
			".*XXXX$",
			"[h-lH-L0-3]+",
		}, wantErr: false},

		{name: "escapes ansi", contents: "\x1b[38;5;140mfoo\x1b[0m bar", pattern: []string{
			"foo",
			"foo bar",
			"^foo\\sbar$",
			"bar",
		}, wantErr: false},

		{name: "escapes ansi complex", contents: "\u001B[0m\u001B[4m\u001B[42m\u001B[31mfoo\u001B[39m\u001B[49m\u001B[24mfoo\u001B[0m", pattern: []string{
			"^foofoo$",
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMimic(WithIdleDuration(5 * time.Millisecond))
			assert.NoError(t, err)

			_, err = m.WriteString(tt.contents)
			assert.NoError(t, err)

			if err := m.ContainsPattern(tt.pattern...); (err != nil) != tt.wantErr {
				t.Errorf("ContainsPattern() error = %#v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMimic_ContainsString(t *testing.T) {
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
			m, err := NewMimic(WithIdleDuration(5 * time.Millisecond))
			assert.NoError(t, err)

			_, err = m.WriteString(tt.contents)
			assert.NoError(t, err)

			if err := m.ContainsString(tt.contains...); (err != nil) != tt.wantErr {
				t.Errorf("ContainsString() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
