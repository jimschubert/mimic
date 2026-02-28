package mimic

import (
	"fmt"
	"strings"
)

// PatternError describes a failure to match one or more patterns against
// terminal contents returned from a Mimic.
type PatternError struct {
	Contents       string
	FailedPatterns []string
}

func (p PatternError) Error() string {
    return fmt.Sprintf("contents failed to match %d pattern(s): %v", len(p.FailedPatterns), strings.Join(p.FailedPatterns, ", "))
}
