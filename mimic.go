package mimic

import (
	"bytes"
	"context"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/Netflix/go-expect"
	creakpty "github.com/creack/pty"
	"github.com/hinshun/vt10x"
	"github.com/jimschubert/mimic/internal"
)

type mimicOpt struct {
	w              io.Writer
	in             io.Reader
	maxIdleTimeout time.Duration
	idleDuration   time.Duration
	rows           int
	columns        int
	pipeFromOS     bool
}

// Option extends functionality of Mimic via functional options.
// see WithOutput, WithStdout, WithSize
type Option func(*mimicOpt)

// WithIdleTimeout defines the timeout period for mimic operations which wait for the terminal to become idle
func WithIdleTimeout(timeout time.Duration) Option {
	return func(opt *mimicOpt) {
		opt.maxIdleTimeout = timeout
	}
}

// WithIdleDuration defines the duration required for mimic to consider the terminal idle
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

// WaitAsync fires an event after wait duration
func (m *Mimic) WaitAsync(wait time.Duration) chan<- struct{} {
	result := make(chan<- struct{}, 1)
	time.AfterFunc(wait, func() {
		result <- struct{}{}
	})
	return result
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

// Read reads bytes from the underlying terminal
// Fulfills the io.Reader interface.
func (m *Mimic) Read(p []byte) (n int, err error) {
	return m.console.Tty().Read(p)
}

// Close causes any underlying emulation to close
// Fulfills the io.Closer interface.
func (m *Mimic) Close() (err error) {
	return m.console.Close()
}

// ViewMatches determines if the emulated terminal's view matches specified string. A "view" takes into account terminal row/columns.
func (m *Mimic) ViewMatches(str string) bool {
	terminalContents := bytes.NewBufferString(m.terminal.String())
	matcher := internal.PlainStringMatcher{
		S: str,
	}
	return matcher.Match(terminalContents)
}

// ContainsPattern determines if the emulated terminal's view contains one or more specified patterns
func (m *Mimic) ContainsPattern(pattern ...string) error {
	var regexes []*regexp.Regexp
	for _, p := range pattern {
		re := regexp.MustCompile(p)
		regexes = append(regexes, re)
	}

	// note: we don't use go-expect's Regexp matcher here because it can invoke multiple times on the buffer
	_, err := m.console.Expect(expect.WithTimeout(m.maxIdleWait), func(opts *expect.ExpectOpts) error {
		opts.Matchers = append(opts.Matchers, &internal.FlushMatcher{})
		return nil
	})
	if err != nil {
		return err
	}

	v := Viewer{Mimic: m, StripAnsi: true, Trim: true}
	contents := v.String()
	failed := make([]string, 0)
	for _, regex := range regexes {
		if !regex.MatchString(contents) {
			failed = append(failed, regex.String())
		}
	}

	if len(failed) == 0 {
		return nil
	}

	return PatternError{FailedPatterns: failed, Contents: contents}
}

// ContainsString determines if the emulated terminal's view contains one or more specified strings
func (m *Mimic) ContainsString(str ...string) error {
	_, err := m.console.Expect(expect.WithTimeout(m.maxIdleWait), internal.String(str...))
	return err
}

// Tty provides the underlying tty required for interacting with this console
func (m *Mimic) Tty() *os.File {
	return m.console.Tty()
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
		columns:        132,
		rows:           24,
		maxIdleTimeout: 5 * time.Second,
		idleDuration:   250 * time.Millisecond,
	}

	for _, opt := range opts {
		opt(o)
	}

	consoleOptions := make([]expect.ConsoleOpt, 0)

	stdIn := make([]io.Reader, 0)
	if o.in != nil {
		stdIn = append(stdIn, o.in)
	}

	stdOut := make([]io.Writer, 0)
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

	terminal := vt10x.New(
		vt10x.WithWriter(tty),
		vt10x.WithSize(o.columns, o.rows),
	)

	consoleOptions = append(consoleOptions, expect.WithSendObserver(func(msg string, num int, err error) {
		if err == nil && num > 0 {
			_, _ = terminal.Write([]byte(msg))
		}
	}))

	c, err := expect.NewConsole(consoleOptions...)

	if err != nil {
		return nil, err
	}

	return &Mimic{
		console:      c,
		terminal:     terminal,
		maxIdleWait:  o.maxIdleTimeout,
		idleDuration: o.idleDuration,
	}, nil
}

func isDebugEnabled() bool {
	if val, ok := os.LookupEnv("DEBUG"); ok {
		debug, _ := strconv.ParseBool(val)
		return debug
	}

	return false
}

var (
	// compile-time contracts (promises made to consumers)
	_ io.ReadWriteCloser = (*Mimic)(nil)
)
