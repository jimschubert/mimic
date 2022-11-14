package mimic_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/jimschubert/mimic"
)

func ExampleMimic_ContainsString() {
	m, _ := mimic.NewMimic(
		mimic.WithSize(24, 30),
		mimic.WithIdleTimeout(10*time.Millisecond),
	)

	text := strings.Repeat("Hi", 16)
	_, _ = m.WriteString(text)

	if err := m.ExpectString(text); err != nil {
		println("The text should have wrapped!")
	}

	if err := m.ExpectString(strings.Repeat("Hi", 15)); err != nil {
		fmt.Printf("Found: %s\n\n", strings.Repeat("Hi", 15))
	}

	formatted := mimic.Viewer{Mimic: m, StripAnsi: true, Trim: true}
	fmt.Printf("Formatted View (30 columns):\n%s\n", formatted.String())

	// Output: Found: HiHiHiHiHiHiHiHiHiHiHiHiHiHiHi
	//
	// Formatted View (30 columns):
	// HiHiHiHiHiHiHiHiHiHiHiHiHiHiHi
	// Hi
}
