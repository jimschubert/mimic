package mimic

import (
	"strings"

	"github.com/jimschubert/stripansi"
)

// Viewer is a utility for providing a String function on a mimic value.
// This is intentionally separated from mimic.Mimic to allow for multiple outputs
// for a single mimic, and to remove any confusion about what String might refer to.
type Viewer struct {
	Mimic     *Mimic
	StripAnsi bool
	Trim      bool
}

// String provides the full underlying dump of the terminal's view.
func (v *Viewer) String() string {
	if v.Mimic == nil {
		return ""
	}

	result := v.Mimic.terminal.String()
	if v.Trim {
		result = strings.TrimSpace(result)
	}

	if v.StripAnsi {
		result = stripansi.String(result)
	}

	return result
}
