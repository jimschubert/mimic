package internal

import (
	"bytes"
	"io"
	"io/fs"
	"regexp"
	"strings"

	"github.com/Netflix/go-expect"
	"github.com/jimschubert/stripansi"
)

// FlushMatcher fulfills the matcher interface, allowing for a flush of go-expect's internal buffer (forces write to vtx10 terminal).
// Functionally, it matches nothing until the end of the stream or a timeout, whichever comes first
type FlushMatcher struct{}

func (f FlushMatcher) Match(v interface{}) bool {
	if err, ok := v.(error); ok {
		if pathError, isReadErr := err.(*fs.PathError); isReadErr {
			if pathError.Op == "read" && pathError.Err.Error() == "i/o timeout" {
				// we've flushed as much as we can and hit a read timeout
				return true
			}
		}
	}
	return false
}

func (f FlushMatcher) Criteria() interface{} {
	return struct{}{}
}

// EOFMatcher matches if the character evaluated is io.EOF
type EOFMatcher struct{}

func (e EOFMatcher) Match(v interface{}) bool {
	err, ok := v.(error)
	if !ok {
		// not an error
		return false
	}
	return err == io.EOF
}

func (e EOFMatcher) Criteria() interface{} {
	return io.EOF
}

// AnyMatcher collects multiple matchers to be evaluated as a single unit via Console.Expect
type AnyMatcher struct {
	Matchers []expect.Matcher
}

func (a AnyMatcher) Match(v interface{}) bool {
	for _, matcher := range a.Matchers {
		if matcher.Match(v) {
			return true
		}
	}
	return false
}

func (a AnyMatcher) Criteria() interface{} {
	var criterias []interface{}
	for _, matcher := range a.Matchers {
		criterias = append(criterias, matcher.Criteria())
	}
	return criterias
}

// PlainStringMatcher fulfills the Matcher interface against strings without ansi codes
// This is nearly the same as https://github.com/Netflix/go-expect/blob/73e0943537d2ba88bdf3f6acec79ca2de1d059df/expect_opt.go#L160
// but differs in that it also escapes ANSI in the buffer to match against plain text
type PlainStringMatcher struct {
	S string
}

func (w PlainStringMatcher) Match(v interface{}) bool {
	buf, ok := v.(*bytes.Buffer)
	if !ok {
		return false
	}
	if strings.Contains(stripansi.String(buf.String()), w.S) {
		return true
	}
	return false
}

func (w PlainStringMatcher) Criteria() interface{} {
	return w.S
}

// RegexpMatcher fulfills the Matcher interface to match Regexp against a given
// bytes.Buffer.
// This is nearly the same as https://github.com/Netflix/go-expect/blob/73e0943537d2ba88bdf3f6acec79ca2de1d059df/expect_opt.go#L181
// but differs in that it also escapes ANSI in the buffer to match against plain text
type RegexpMatcher struct {
	Re *regexp.Regexp
}

func (rm *RegexpMatcher) Match(v interface{}) bool {
	buf, ok := v.(*bytes.Buffer)
	if !ok {
		return false
	}
	stripped := stripansi.Bytes(buf.Bytes())
	return rm.Re.Match(stripped)
}

func (rm *RegexpMatcher) Criteria() interface{} {
	return rm.Re
}

// String adds an Expect condition to exit if the content read from Console'S
// tty contains any of the given strings. Matched against Console contents with ansi characters stripped.
func String(strs ...string) expect.ExpectOpt {
	return func(opts *expect.ExpectOpts) error {
		for _, str := range strs {
			opts.Matchers = append(opts.Matchers, &PlainStringMatcher{
				S: str,
			})
		}
		return nil
	}
}

// Regexp adds an Expect condition to exit if the content read from Console'S
// tty matches the given Regexp.
func Regexp(res ...*regexp.Regexp) expect.ExpectOpt {
	return func(opts *expect.ExpectOpts) error {
		for _, re := range res {
			opts.Matchers = append(opts.Matchers, &RegexpMatcher{
				Re: re,
			})
		}
		return nil
	}
}
