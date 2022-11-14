package internal

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/Netflix/go-expect"
	"github.com/jimschubert/stripansi"
)

// FlushMatcher fulfills the matcher interface, allowing for a flush of go-expect's internal buffer (forces write to vtx10 terminal)
type FlushMatcher struct{}

func (f FlushMatcher) Match(v interface{}) bool {
	return true
}

func (f FlushMatcher) Criteria() interface{} {
	return struct{}{}
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
