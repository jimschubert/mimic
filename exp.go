package mimic

import (
	"errors"

	"github.com/Netflix/go-expect"
	"github.com/hinshun/vt10x"
)

// An Experimental contract which can be changed or removed at any time.
// This is intended for use by users for experimentation purposes only.
type Experimental interface {
	// Console provides access to the underlying expect.Console
	Console() (expect.Console, error)
	// Terminal provides access to the underlying vt10x.Terminal
	Terminal() (vt10x.Terminal, error)
}

type exp Mimic

// Console provides access to the underlying expect.Console
func (e exp) Console() (expect.Console, error) {
	if e.console == nil {
		return expect.Console{}, errors.New("console is uninitialized")
	}
	return *e.console, nil
}

// Terminal provides access to the underlying vt10x.Terminal
func (e exp) Terminal() (vt10x.Terminal, error) {
	return e.terminal, nil
}
