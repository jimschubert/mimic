package mimic

import (
	"fmt"
	"strings"
)

type PatternError struct {
	Contents       string
	FailedPatterns []string
}

func (p PatternError) Error() string {
	var suffix string
	count := len(p.FailedPatterns)
	if count > 0 {
		suffix = "s"
	}

	return fmt.Sprintf("contents failed to match %d pattern%s: %v", count, suffix, strings.Join(p.FailedPatterns, ", "))
}
