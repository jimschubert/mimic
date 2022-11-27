package mimic

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Netflix/go-expect"
	creakpty "github.com/creack/pty"
	"github.com/hinshun/vt10x"
	"github.com/jimschubert/mimic/internal"
)

const (
	// DefaultColumns for the underlying view-based terminal's column count (i.e. width)
	DefaultColumns = 132
	// DefaultRows for the underlying view-based terminal's row count (i.e. height)
	DefaultRows = 24
	// DefaultIdleTimeout when the underlying terminal is idle (i.e. fails to match an expectation), used by functions
	// such as Mimic.ExpectString, Mimic.ContainsString, Mimic.ExpectPattern, and Mimic.ContainsPattern
	DefaultIdleTimeout = 250 * time.Millisecond
	// DefaultFlushTimeout for mimic's flush operation. Mimic will invoke flush only if there are outstanding operations
	// from Mimic.Write or Mimic.WriteString.
	DefaultFlushTimeout = 25 * time.Millisecond
	// DefaultIdleDuration for mimic to consider the terminal idle via Mimic.WaitForIdle.
	DefaultIdleDuration = 100 * time.Millisecond
)

type mimicOpt struct {
	w              io.Writer
	in             io.Reader
	maxIdleTimeout time.Duration
	idleDuration   time.Duration
	flushTimeout   time.Duration
	rows           int
	columns        int
	pipeFromOS     bool
}

// Option extends functionality of Mimic via functional options.
// see WithOutput, WithStdout, WithSize
type Option func(*mimicOpt)

// WithFlushTimeout defines the timeout for mimic's flush operation. Mimic will invoke flush only if there
// are outstanding operations from Mimic.Write or Mimic.WriteString.
func WithFlushTimeout(timeout time.Duration) Option {
	return func(opt *mimicOpt) {
		opt.flushTimeout = timeout
	}
}

// WithIdleTimeout defines the timeout period for mimic operations which wait for the terminal to become idle
func WithIdleTimeout(timeout time.Duration) Option {
	return func(opt *mimicOpt) {
		opt.maxIdleTimeout = timeout
	}
}

// WithIdleDuration defines the duration required for mimic to consider the terminal idle via Mimic.WaitForIdle.
func WithIdleDuration(duration time.Duration) Option {
	return func(opt *mimicOpt) {
		opt.idleDuration = duration
	}
}

// WithOutput writes a copy of emulated console output to w
// Not compatible with WithStdout
func WithOutput(w io.Writer) Option {
	return func(opt *mimicOpt) {
		if opt.w != io.Discard {
			panic("Mimic's writer can only be set once")
		}
		opt.w = w
	}
}

// WithInput accepts input from r
func WithInput(r io.Reader) Option {
	return func(opt *mimicOpt) {
		opt.in = r
	}
}

// WithPipeFromOS determines whether standard os streams should be included in the pseudo terminal
func WithPipeFromOS() Option {
	return func(opt *mimicOpt) {
		opt.pipeFromOS = true
	}
}

// WithSize defines the size of the emulated terminal
func WithSize(rows, columns int) Option {
	return func(opt *mimicOpt) {
		opt.rows = rows
		opt.columns = columns
	}
}

// Mimic is a utility for mimicking operations on a pseudo terminal
type Mimic struct {
	console      *expect.Console
	terminal     vt10x.Terminal
	maxIdleWait  time.Duration
	idleDuration time.Duration
	flushTimeout time.Duration
	Experimental Experimental
}

// WaitForIdle causes the emulated terminal to spin, waiting the terminal output to "stabilize" (i.e. no writes are occurring)
func (m *Mimic) WaitForIdle(ctx context.Context) error {
	done := make(chan struct{})
	timeoutContext, cancel := context.WithTimeout(ctx, m.maxIdleWait)
	defer cancel()
	go func() {
		defer close(done)
		var coord vt10x.Cursor
		emptyCoord := vt10x.Cursor{}

		started := time.Now()
		for {
			if timeoutContext.Err() != nil {
				// context is completed before we begin iteration
				return
			}

			if coord != m.terminal.Cursor() {
				coord = vt10x.Cursor{}
				started = time.Now()
			}

			if coord != emptyCoord && time.Now().Sub(started) >= m.idleDuration {
				done <- struct{}{}
				return
			}

			coord = m.terminal.Cursor()
			time.Sleep(1 * time.Millisecond)
		}
	}()

	select {
	case <-timeoutContext.Done():
		// we didn't stabilize :(
		return timeoutContext.Err()
	case <-done:
		return nil
	}
}

// WriteString writes a value to the underlying terminal
func (m *Mimic) WriteString(str string) (int, error) {
	return m.console.Send(str)
}

// Write writes a value to the underlying terminal.
// Fulfills the io.Writer interface.
func (m *Mimic) Write(b []byte) (int, error) {
	return m.WriteString(string(b))
}

// Read bytes from the underlying terminal
// Fulfills the io.Reader interface.
func (m *Mimic) Read(p []byte) (n int, err error) {
	return m.console.Tty().Read(p)
}

// Close causes any underlying emulation to close.
// Fulfills the io.Closer interface.
func (m *Mimic) Close() (err error) {
	return m.console.Close()
}

// Flush (or attempt to flush) any pending writes done via Write or WriteString.
func (m *Mimic) Flush() error {
	_, err := m.console.Expect(expect.WithTimeout(m.flushTimeout), func(opts *expect.ExpectOpts) error {
		opts.Matchers = append(opts.Matchers, &internal.AnyMatcher{Matchers: []expect.Matcher{
			&internal.EOFMatcher{},
			&internal.FlushMatcher{},
		}})
		return nil
	})

	return err
}

// ContainsString determines if the emulated terminal's view matches specified string. A "view" takes into account terminal row/columns.
// Terminal contents are stripped of ANSI escape characters and trimmed.
func (m *Mimic) ContainsString(str ...string) bool {
	// note: we don't use go-expect's Regexp matcher here because it can invoke multiple times on the buffer
	// instead, we Flush which writes all runes to the terminal view, and check regexes against that
	err := m.Flush()
	if err != nil {
		if isDebugEnabled() {
			_, _ = fmt.Fprintf(os.Stderr, "[Error]: ContainsString: %v\n", err)
		}
		return false
	}

	v := Viewer{Mimic: m, StripAnsi: true, Trim: true}
	contents := v.String()

	failed := 0
	terminalContents := bytes.NewBufferString(contents)

	for _, s := range str {
		matcher := internal.PlainStringMatcher{
			S: s,
		}
		if !matcher.Match(terminalContents) {
			failed += 1
		}
	}
	return failed == 0
}

// ContainsPattern determines if the emulated terminal's view contains one or more specified patterns.
// Patterns are evaluated against formatted terminal contents, stripped of ANSI escape characters and trimmed.
func (m *Mimic) ContainsPattern(pattern ...string) bool {
	var regexes []*regexp.Regexp
	for _, p := range pattern {
		re := regexp.MustCompile(p)
		regexes = append(regexes, re)
	}

	// note: we don't use go-expect's Regexp matcher here because it can invoke multiple times on the buffer
	// instead, we Flush which writes all runes to the terminal view, and check regexes against that
	err := m.Flush()
	if err != nil {
		if isDebugEnabled() {
			_, _ = fmt.Fprintf(os.Stderr, "[Error]: ContainsPattern: %v\n", err)
		}
		return false
	}

	v := Viewer{Mimic: m, StripAnsi: true, Trim: true}
	contents := v.String()
	failed := make([]string, 0)
	for _, regex := range regexes {
		if !regex.MatchString(contents) {
			failed = append(failed, regex.String())
		}
	}

	if len(pattern) > 0 && len(failed) == 0 {
		return true
	}

	if isDebugEnabled() {
		_, _ = fmt.Fprintf(os.Stderr, "[Error]: ContainsPattern failed on: %v\n", strings.Join(failed, ","))
	}

	return false
}

// ExpectPattern waits for the emulated terminal's view to contain one or more specified patterns
func (m *Mimic) ExpectPattern(pattern ...string) error {
	var regexes []*regexp.Regexp
	for _, p := range pattern {
		re := regexp.MustCompile(p)
		regexes = append(regexes, re)
	}
	_, err := m.console.Expect(expect.WithTimeout(m.maxIdleWait), internal.Regexp(regexes...))
	return err
}

// ExpectString waits for the emulated terminal's view to contain one or more specified strings
func (m *Mimic) ExpectString(str ...string) error {
	_, err := m.console.Expect(expect.WithTimeout(m.maxIdleWait), internal.String(str...))
	return err
}

// NoMoreExpectations signals the underlying buffer to finish writing bytes to the underlying pseudo-terminal.
func (m *Mimic) NoMoreExpectations() error {
	// We flush here because ExpectEOF can sometimes "hang" if there are no Expect interactions prior to calling it.
	err := m.Flush()
	if err != nil {
		if isDebugEnabled() {
			_, _ = fmt.Fprintf(os.Stderr, "[Error]: NoMoreExpectations: %v", err)
		}
		return err
	}

	_, err = m.console.ExpectEOF()
	return err
}

// Tty provides the underlying tty required for interacting with this console
func (m *Mimic) Tty() *os.File {
	return m.console.Tty()
}

// Fd file descriptor of underlying pty.
func (m *Mimic) Fd() uintptr {
	return m.console.Fd()
}

// NewMimic creates a Mimic, which emulates a pseudo terminal device and provides
// utility functions for inputs/assertions/expectations upon it
func NewMimic(opts ...Option) (*Mimic, error) {
	pty, tty, err := creakpty.Open()
	if err != nil {
		return nil, err
	}

	o := &mimicOpt{
		w:              io.Discard,
		columns:        DefaultColumns,
		rows:           DefaultRows,
		maxIdleTimeout: DefaultIdleTimeout,
		flushTimeout:   DefaultFlushTimeout,
		idleDuration:   DefaultIdleDuration,
	}

	for _, opt := range opts {
		opt(o)
	}

	consoleOptions := make([]expect.ConsoleOpt, 0)

	terminal := vt10x.New(
		vt10x.WithWriter(tty),
		vt10x.WithSize(o.columns, o.rows),
	)

	stdIn := make([]io.Reader, 0)
	stdIn = append(stdIn, pty)
	if o.in != nil {
		stdIn = append(stdIn, o.in)
	}

	stdOut := make([]io.Writer, 0)
	stdOut = append(stdOut, terminal)
	if o.w != nil {
		stdOut = append(stdOut, o.w)
	}

	if o.pipeFromOS {
		stdIn = append(stdIn, os.Stdin)
		stdOut = append(stdOut, os.Stdout)
	}

	consoleOptions = append(consoleOptions, expect.WithStdin(stdIn...))
	consoleOptions = append(consoleOptions, expect.WithStdout(stdOut...))
	consoleOptions = append(consoleOptions, expect.WithCloser(pty, tty))

	if isDebugEnabled() {
		consoleOptions = append(consoleOptions, expect.WithLogger(log.New(os.Stderr, "mimic: ", 0)))
	}

	c, err := expect.NewConsole(consoleOptions...)

	if err != nil {
		return nil, err
	}

	m := Mimic{
		console:      c,
		terminal:     terminal,
		maxIdleWait:  o.maxIdleTimeout,
		idleDuration: o.idleDuration,
		flushTimeout: o.flushTimeout,
	}

	m.Experimental = exp(m)

	return &m, nil
}

func isDebugEnabled() bool {
	if val, ok := os.LookupEnv("DEBUG"); ok {
		debug, _ := strconv.ParseBool(val)
		return debug
	}

	return false
}

// for file-based Stdout
type fileWriter interface {
	io.Writer
	Fd() uintptr
}

// for file-based Stdin
type fileReader interface {
	io.Reader
	Fd() uintptr
}

var (
	// compile-time contracts (promises made to consumers)
	_ io.ReadWriteCloser = (*Mimic)(nil)
	_ fileWriter         = (*Mimic)(nil)
	_ fileReader         = (*Mimic)(nil)
)
