# mimic - testing terminal interactions

[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/jimschubert/mimic?color=%23007D9C&label=github.com%2Fjimschubert%2Fmimic&logo=go&logoColor=white)](#installing)
[![Go Reference](https://pkg.go.dev/badge/github.com/jimschubert/mimic.svg)](https://pkg.go.dev/github.com/jimschubert/mimic)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/jimschubert/mimic?color=%23007D9C&logo=go&label&logoColor=white)

[![GitHub](https://img.shields.io/github/license/jimschubert/mimic?color=%23007D9C&logo=apache&label=LICENSE)](./LICENSE)
[![Go Build](https://github.com/jimschubert/mimic/actions/workflows/build.yml/badge.svg)](https://github.com/jimschubert/mimic/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jimschubert/mimic)](https://goreportcard.com/report/github.com/jimschubert/mimic)

Mimic aims to provide a simple and clean contract for interacting with terminal contents while unit testing.

The idea arose from use of [go-survey/survey](https://github.com/go-survey/survey) and interest in using something like [charmbracelet/vhs](https://github.com/charmbracelet/vhs) for integration testing of a CLI.
A well-crafted applications can use mimic for integration testing of CLIs with full capability to mock dependencies. This differs from integration testing with tools like vhc because mimic runs in-process.

A goal of this project is to simplify testing CLIs using `survey` by providing a test suite which reduces test boilerplate and adheres to testify's suite interface(s).

Mimic also provides commands for:

* checking existence of strings in the output buffer
* checking existence of patterns in the output buffer
* checking formatted output according to pseudoterminal configuration

## Installing

First, get the latest version of mimic:

```shell
go get -u github.com/jimschubert/mimic@latest
```

Then, import mimic:

```go
import "github.com/jimschubert/mimic"
```

## Usage

Refer to [documentation](https://godoc.org/github.com/jimschubert/mimic) for general API and usage.

For a real-world example, see [suite_test.go](./suite/suite_test.go) in this repository.

### Expect vs Contains

Mimic works by wrapping both [Netflix/go-expect](https://github.com/Netflix/go-expect) and [hinshun/vt10x](https://github.com/hinshun/vt10x) to provide two APIs which require a brief explanation.


#### Expect

The expect-like APIs (`mimic.ExpectString` and `mimic.ExpectPattern`) watch the stream of stdout for a string or pattern, evaluating your criteria on each new byte written to the stream. Once criteria is met, you can't re-evaluate those same bytes. If these are concerns, consider using `mimic.ContainsString` and `mimic.ContainsPattern` which both flush pending writes (up to the flush timeout) prior to evaluating conditions.

For example, suppose you want to orchestrate the following terminal output:

```
? What is your name? Jim
? What is your github username? jimschubert
```

The following will work in your test:

```go
assert.NoError(t, mimic.ExpectString("What is your name?"))
```

Expecting on multiple conditions for the same line will fail because it's expecting the contents to be written to stdout again. For example, expecting a partial and full line as follows does not work:

```go
assert.NoError(t, mimic.ExpectString("What is your name?"))
mimic.WriteString("Jim")

// can't do this with ExpectString
assert.NoError(t, mimic.ExpectString("? What is your name?"))
```

You could use a pattern for both of these:

```go
assert.NoError(t, mimic.ExpectPattern("What is your.*name"))
mimic.WriteString("Jim")
assert.NoError(t, mimic.ExpectPattern("What is your.*name"))
mimic.WriteString("jimschubert")
```

The above example is contrived to demonstrate a concern with test performance when using ExpectPattern. **The pattern is evaluated against every new byte on the stream.** You could test this locally by adding a log message to RegexpMatcher.Match in this repository. You'd see something like this:

```
mimic: [RegexpMatcher] evaluating: ?
mimic: [RegexpMatcher] evaluating: ? 
mimic: [RegexpMatcher] evaluating: ? W
mimic: [RegexpMatcher] evaluating: ? Wh
mimic: [RegexpMatcher] evaluating: ? Wha
mimic: [RegexpMatcher] evaluating: ? What
mimic: [RegexpMatcher] evaluating: ? What 
mimic: [RegexpMatcher] evaluating: ? What i
mimic: [RegexpMatcher] evaluating: ? What is
mimic: [RegexpMatcher] evaluating: ? What is 
mimic: [RegexpMatcher] evaluating: ? What is y
mimic: [RegexpMatcher] evaluating: ? What is yo
mimic: [RegexpMatcher] evaluating: ? What is you
mimic: [RegexpMatcher] evaluating: ? What is your
mimic: [RegexpMatcher] evaluating: ? What is your 
mimic: [RegexpMatcher] evaluating: ? What is your n
mimic: [RegexpMatcher] evaluating: ? What is your na
mimic: [RegexpMatcher] evaluating: ? What is your nam
mimic: [RegexpMatcher] evaluating: ? What is your name
mimic: [RegexpMatcher] evaluating: ?
```

If you wanted to validate that `What is your name?` is not output twice, expecting via `assert.Error` will run until the timeout period. Suppose you have a timeout of 5 seconds (constructed with `mimic.WithIdleTimeout(5 * time.Second)`). The following `assert.Error` is technically valid, but adds 5 seconds to your test function:

```go
assert.NoError(t, mimic.ExpectString("What is your name?"))
mimic.WriteString("Jim")
assert.Error(t, mimic.ExpectString("What is your name?"), "This condition should succeed after 5 seconds!")
```

Once you're done with all interactions, it's a best practice to invoke `mimic.NoMoreExpectations()`. This flushes remaining bytes to stdout and expects `io.EOF` on the stream.

## Contains

The contains APIs (`mimic.ContainsString` and `mimic.ContainsPattern`) both flush pending writes (up to the flush timeout) prior to evaluating conditions.

For example, suppose you want to orchestrate the following terminal output:

```
? What is your name? Jim
? What is your github username? jimschubert
```

The following will work in your test:

```go
assert.True(t, mimic.ContainsString("What is your name?"))
```

The test case described in the [Expect](#expect) second which failed would work using ContainsString:

```go
assert.True(t, mimic.ContainsString("What is your name?"))
mimic.WriteString("Jim")

// can do this with ContainsString, but not ExpectString
assert.True(t, mimic.ContainsString("? What is your name?"))
```

`ContainsString` and `ContainsPattern` work on the full terminal view. Keep this in mind as the following which works serially in the Expect API works a little differently:

```go
assert.True(t, mimic.ContainsPattern("What is your.*name"))
mimic.WriteString("Jim")

// WARNING: Sill passes for "What is your name?" even if "What is your github username?" is never displayed
assert.True(t, mimic.ContainsPattern("What is your.*name"))
mimic.WriteString("jimschubert")
```

Since `ContainsPattern` works on the entire view contents, each regex passed to the function is evaluated once. If you were to add a logger within the `ContainsPattern` function, you'd see a trace similar to (one log for each of the above assertions):

```
mimic: [ContainsPattern] evaluating: What is your.*name
mimic: [ContainsPattern] evaluating: What is your.*name
```

You can always mix and match these APIs:

```go
assert.True(t, mimic.ContainsPattern("What is your.*name"))
mimic.WriteString("Jim")
assert.NoError(t, mimic.ExpectPattern("What is your.*name"))
mimic.WriteString("jimschubert")
```

**Prefer `ContainsString` or `ExpectString` over pattern based functions where possible.

## License

This project is [licensed](./LICENSE) under Apache 2.0.
