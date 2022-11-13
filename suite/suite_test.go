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

func (m *MyTests) TestMimicWithoutOptions() {
	m.T().Log("Invoked TestMimicWithoutOptions")
	console, err := m.Mimic()
	assert.NoError(m.T(), err, "Parameterless invocation should not produce an error")
	assert.NotNil(m.T(), console, "Mimic instance must should not be nil on errorless construction")
}

func (m *MyTests) TestMimicWithOptions() {
	m.T().Log("Invoked TestMimicWithOptions")
	console, err := m.Mimic(
		mimic.WithIdleDuration(50*time.Millisecond),
		mimic.WithIdleTimeout(1*time.Second),
		mimic.WithPipeFromOS(),
	)

	assert.NoError(m.T(), err, "Standard invocation with options should not produce an error")
	assert.NotNil(m.T(), console, "Mimic instance must should not be nil on errorless construction")
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

	assert.NoError(m.T(), console.ContainsString(full), "Emulated terminal should be %d columns, not %d columns as the written string", terminalWidth, fullWriteWidth)
	assert.Error(m.T(), console.ContainsString(strings.Repeat(character, terminalWidth+1)), "Emulated terminal should be %d columns, but found %d characters", terminalWidth, terminalWidth+1)
	assert.Error(m.T(), console.ContainsString(strings.Repeat(character, terminalWidth)), "Emulated terminal should have wrapped text at %d columns", terminalWidth)

	assert.False(m.T(), console.ViewMatches(full), "underlying terminal is expected to wrap")
	assert.True(m.T(), console.ViewMatches(strings.Repeat(character, terminalWidth)+"\n"+strings.Repeat(character, wrapLength)), "underlying terminal is expected to wrap")
}

func (m *MyTests) TestMimicWaitingForIdle() {
	started := time.Now()
	m.T().Logf("Invoked TestSomethingElse at %s", started.Format(time.RFC822Z))

	console, _ := m.Mimic(
		mimic.WithIdleDuration(50*time.Millisecond),
		mimic.WithIdleTimeout(1*time.Second),
	)

	targetCount := 30

	go func() {
		count := 0
		var writer io.Writer
		writer = console
		for {
			if count >= targetCount {
				// avoid an issue in test runner where missing newline will result in "no tests" found
				//goland:noinspection GoUnhandledErrorResult
				writer.Write([]byte("\n"))
				return
			}
			//goland:noinspection GoUnhandledErrorResult
			writer.Write([]byte("."))
			time.Sleep(1 * time.Millisecond)
			count += 1
		}
	}()

	if err := console.WaitForIdle(context.TODO()); err != nil {
		m.T().Errorf("Waiting for console to stabile failed after %dms", time.Now().Sub(started).Milliseconds())
		m.T().Fail()
		return
	}

	assert.NoError(m.T(), console.ContainsString(strings.Repeat(".", targetCount)), "Console didn't include expected contents… Was: empty")
	assert.Error(m.T(), console.ContainsString(strings.Repeat(".", targetCount+2)), "Console did not include expected contents… Was: empty")
}

func TestMimicOperationsSuite(t *testing.T) {
	test := new(MyTests)
	test.suiteRuntimeDuration = 30 * time.Second
	test.Init(WithMaxRuntime(test.suiteRuntimeDuration))

	suite.Run(t, test)
}
