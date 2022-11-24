/*
Package mimic provides a utility for interacting with console or terminal based applications.

A mimic/Mimic internally constructs two pseudo-terminals: one wrapping go-expect,
and another construct of creak/pty. This allows for either stream-based or view-based inspection of strings/patterns.

The key difference between the two is that stream-based inspections provided by
Mimic.ExpectString and Mimic.ExpectPattern will wait for a configurable amount
of time for any text matching the criteria, then _fail_ if no match is found. The search
criteria passed to these functions is evaluated repeatedly as bytes are written to your
output stream. Keep this in mind, as very complex patterns can be slow. The underlying views are raw pty,
and the output is therefore not formatted as it would be within a terminal.

The view-based inspections provided by Mimic.ContainsString and Mimic.ContainsPattern,
on the other hand, will wait for the bound output stream to complete processing before applying
the search criteria to the entire formatted view. This takes configurable terminal columns/rows
into account. These default to a large standard of 132 columns and 24 rows. Internally, this is implemented
via github.com/hinshun/vt10x.

Usage

A mimic value implements io.ReadWriteCloser and also satisfies the following interfaces:

	 type fileWriter interface {
		io.Writer
		Fd() uintptr
	 }

	 type fileReader interface {
		io.Reader
		Fd() uintptr
	 }

This allows Mimic values to be used in place of Stdin/Stdout/Stderr in most scenarios, including
implementations using github.com/AlecAivazis/survey/v2. For example:

	 package main

	 import (
		"fmt"
		"os"

		"github.com/AlecAivazis/survey/v2"
		"github.com/jimschubert/mimic"
	 )

	 func main() {
		console, _ := mimic.NewMimic()
		answers := struct {
			Name string
			Age  int
		}{}

		go func() {
			// errors ignored for brevity
			console.ExpectString("What is your name?")
			console.WriteString("Tom\n")
			console.ExpectString("How old are you?")
			console.WriteString("20\n")
			console.ExpectString("Tom", "20")
			if !console.ContainsString("What is your name?", "How old are you?", "Tom", "20") {
				panic("My answers weren't displayed!")
			}
			_ = console.NoMoreExpectations()
		}()

		_ = survey.Ask([]*survey.Question{
			{Name: "name", Prompt: &survey.Input{Message: "What is your name?"}},
			{Name: "age", Prompt: &survey.Input{Message: "How old are you?"}},
		}, &answers,
			survey.WithStdio(console.Tty(), console.Tty(), console.Tty()),
		)
		fmt.Fprintf(os.Stdout, "%s is %d.\n", answers.Name, answers.Age)
	 }

Notice in the above example that all expectations should be invoked asynchronously from the thread being instrumented.

Testing

Mimic provides a Suite based on github.com/stretchr/testify/suite which allows creation of a new mimic per test, or
a suite-level mimic can be created for more advanced scenarios. Embed suite.Suite into a test struct, then add Test* functions
to implement your tests. Follow testify's documentation for more. Here's an slimmed-down example from mimic's own tests:

	  package suite

	  import (
		"context"
		"io"
		"strings"
		"testing"
		"time"

		"github.com/jimschubert/mimic"
		"github.com/stretchr/testify/assert"
		"github.com/stretchr/testify/suite"
	  )

	  type MyTests struct {
		Suite
		suiteRuntimeDuration time.Duration
	  }

	  func (m *MyTests) SetupSuite() {
		assert.Greaterf(m.T(), m.suiteRuntimeDuration, 0*time.Second, "Suite runtime must be marked as more than 0 units of time")
	  }

	  func (m *MyTests) TestMimicWriteRead() {
		m.T().Log("Invoked TestMimicWithOptions")
		terminalWidth := 80
		wrapLength := 20
		console, err := m.Mimic(
			mimic.WithIdleDuration(25*time.Millisecond),
			mimic.WithIdleTimeout(1*time.Second),
			mimic.WithSize(24, terminalWidth),
		)

		assert.NoError(m.T(), err, "Standard invocation with options should not produce an error")
		assert.NotNil(m.T(), console, "Mimic instance must should not be nil on errorless construction")

		character := "X"
		fullWriteWidth := terminalWidth + wrapLength
		full := strings.Repeat(character, fullWriteWidth)
		written, err := console.WriteString(full)

		assert.NoError(m.T(), err, "pty should have allowed the write!")
		assert.Equal(m.T(), written, fullWriteWidth, "pty should have written all bytes!")

		assert.NoError(m.T(), console.ExpectString(full), "Emulated terminal should be %d columns, not %d columns as the written string", terminalWidth, fullWriteWidth)
		assert.Error(m.T(), console.ExpectString(strings.Repeat(character, terminalWidth+1)), "Emulated terminal should be %d columns, but found %d characters", terminalWidth, terminalWidth+1)
		assert.Error(m.T(), console.ExpectString(strings.Repeat(character, terminalWidth)), "Emulated terminal should have wrapped text at %d columns", terminalWidth)

		assert.False(m.T(), console.ContainsString(full), "underlying terminal is expected to wrap")
		assert.True(m.T(), console.ContainsString(strings.Repeat(character, terminalWidth)+"\n"+strings.Repeat(character, wrapLength)), "underlying terminal is expected to wrap")
	  }

	  func TestMimicOperationsSuite(t *testing.T) {
		test := new(MyTests)
		test.suiteRuntimeDuration = 30 * time.Second
		test.Init(WithMaxRuntime(test.suiteRuntimeDuration))

		suite.Run(t, test)
	  }
*/
package mimic
