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

func ExampleViewer_String() {
	m, _ := mimic.NewMimic(
		mimic.WithSize(24, 80),
		mimic.WithIdleTimeout(300*time.Millisecond),
	)

	text := "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Sed vulputate odio ut enim blandit volutpat maecenas volutpat."
	_, _ = m.WriteString(text)
	_ = m.NoMoreExpectations()
	formatted := mimic.Viewer{Mimic: m, StripAnsi: true, Trim: true}
	fmt.Printf("%s\n", formatted.String())

	// Output:
	// Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor i
	// ncididunt ut labore et dolore magna aliqua. Sed vulputate odio ut enim blandit v
	// olutpat maecenas volutpat.
}
