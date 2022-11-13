package suite

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"
	"time"

	"github.com/jimschubert/mimic"
)

//goland:noinspection GoUnusedGlobalVariable
var (
	suiteTestPattern = regexp.MustCompile(`\(\*?(?P<suiteName>[a-zA-Z_0-9]+)\)\.(?P<testName>[a-zA-Z_0-9]+)\b`)

	NoOptions []SuiteOption
)

type opt func(b *Suite)

//goland:noinspection GoNameStartsWithPackageName
type SuiteOption opt

// WithMaxRuntime signals to the suite how long the user expects the suite to run (at a maximum). This allows us to
// mark a suite as a failure if it is taking far too long. It also optionally allows us to fail fast, preventing all other
// tests from invoking (i.e. timeout tests - useful for CI where test runtime can increase costs).
func WithMaxRuntime(duration time.Duration) SuiteOption {
	return func(b *Suite) {
		b.maxRuntime = duration
		timeoutCtx, cancelFn := context.WithTimeout(b.ctx, b.maxRuntime)
		b.ctx = timeoutCtx
		go func() {
			defer cancelFn()
			for {
				select {
				case <-b.quit:
					cancelFn()
				case <-b.ctx.Done():
					if errors.Is(b.ctx.Err(), context.DeadlineExceeded) {
						b.T().Log(string(debug.Stack()))
						b.T().Errorf("Suite exceeded max runtime of %v!", b.maxRuntime)
						b.quit <- struct{}{}
					}
					return
				}
			}
		}()
	}
}

type Suite struct {
	t          *testing.T
	testCases  map[string]*testCase
	suiteMimic *mimic.Mimic
	maxRuntime time.Duration

	ctx  context.Context
	quit chan struct{}
	once sync.Once
}

func (b *Suite) initialize() {
	b.testCases = make(map[string]*testCase)
	b.ctx = context.Background()
	b.quit = make(chan struct{})
}

// SuiteOptions allows user extension of options in a consistent manner.
// Some IDEs allow for running tests individually, which may skip any Init logic defined in the test which originally
// constructs and initializes the suite. Preferred usage is to define options via SuiteOptions to ensure these options
// are applied consistently across tests in the suite, regardless of how the user invokes the test.
func (b *Suite) SuiteOptions() []SuiteOption {
	return []SuiteOption{
		WithMaxRuntime(5 * time.Minute),
	}
}

// T obtains a reference to the underlying testing.T used by the suite
func (b *Suite) T() *testing.T {
	return b.t
}

// SetT is intended for use internally for setting the generated testing.T for a given test
func (b *Suite) SetT(t *testing.T) {
	b.t = t
}

// SetSuiteMimic allows for a suite-level mimic reference.
// This can be helpful for complex suites applying test cases across a global pty. However, such tests can
// be flaky. suite.Suite is built upon testify's Suite which guarantees serial invocation, which helps.
// Use this sparingly.
func (b *Suite) SetSuiteMimic(m *mimic.Mimic) {
	b.suiteMimic = m
}

func (b *Suite) key(suiteName string, testName string) string {
	return fmt.Sprintf("%s_%s", suiteName, testName)
}

// BeforeTest applies test-level preparations prior to running a test found within the suite
func (b *Suite) BeforeTest(suiteName string, testName string) {
	key := b.key(suiteName, testName)
	b.testCases[key] = &testCase{
		TestName: testName,
		mimic:    b.suiteMimic,
	}
}

// AfterTest applies test-level cleanup after running a test found within the suite
func (b *Suite) AfterTest(suiteName string, testName string) {
	key := b.key(suiteName, testName)
	v := b.testCases[key]
	if v == nil {
		return
	}

	// todo: reset suite mimic's console after each test?
	if b.suiteMimic == nil && v.mimic != nil {
		_ = v.mimic.Close()
	}
}

// SetupTestSuite applies suite-level setup logic
func (b *Suite) SetupTestSuite() {
	options := b.SuiteOptions()
	if options != nil {
		b.Init(options...)
		return
	}
	// ensure we're initialized, even if user returns nil suite options
	b.once.Do(b.initialize)
}

// TearDownSuite applies suite-level teardown logic
func (b *Suite) TearDownSuite() {
	if b.suiteMimic != nil {
		defer func(suiteMimic *mimic.Mimic) {
			_ = suiteMimic.Close()
		}(b.suiteMimic)
	}
	if b.quit != nil {
		defer close(b.quit)
	}
}

func (b *Suite) caller() string {
	counter, _, _, success := runtime.Caller(2)
	if !success {
		return ""
	}

	// e.g. github.com/jimschubert/mimic.(*MyTests).TestSomethingElse
	invoker := runtime.FuncForPC(counter).Name()
	var suiteName, testName string
	for _, match := range suiteTestPattern.FindAllStringSubmatch(invoker, -1) {
		for groupIdx, group := range match {
			if groupIdx == 1 {
				suiteName = group
			} else if groupIdx == 2 {
				testName = group
			}
		}
	}

	return b.key(suiteName, testName)
}

// Mimic constructs a new mimic for the given opts, which is specific to the current test case.
func (b *Suite) Mimic(opts ...mimic.Option) (*mimic.Mimic, error) {
	key := b.caller()
	if key == "" {
		return nil, errors.New("unable to determine name of calling test function")
	}

	if tc, ok := b.testCases[key]; ok && tc.mimic != nil {
		return tc.mimic, nil
	}

	var err error
	b.testCases[key].mimic, err = mimic.NewMimic(opts...)
	return b.testCases[key].mimic, err
}

// Init applies suite options to initialize the test suite
func (b *Suite) Init(opts ...SuiteOption) {
	b.once.Do(b.initialize)
	for _, option := range opts {
		option(b)
	}
}

type testCase struct {
	TestName string
	mimic    *mimic.Mimic
}
