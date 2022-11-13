package mimic

import (
	"strings"

	"github.com/jimschubert/stripansi"
)

type Viewer struct {
	Mimic     *Mimic
	StripAnsi bool
	Trim      bool
}

func (v *Viewer) String() string {
	if v.Mimic == nil {
		return ""
	}

	var altered string
	original := v.Mimic.terminal.String()
	if v.Trim {
		altered = strings.TrimSpace(original)
	}

	if v.StripAnsi {
		altered = stripansi.String(altered)
	}

	return altered
}
