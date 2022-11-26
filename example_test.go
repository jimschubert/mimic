package mimic_test

import (
	"fmt"
	"strings"
	"time"

	"github.com/jimschubert/mimic"
)

func ExampleMimic_ContainsString() {
	columns := 26
	m, _ := mimic.NewMimic(
		mimic.WithSize(24, columns),
		mimic.WithFlushTimeout(75*time.Millisecond),
		mimic.WithIdleDuration(50*time.Millisecond),
	)

	// create three rows of text…
	for row := 1; row <= 3; row++ {
		for i := 'a'; i <= 'z'; i++ {
			_, _ = m.WriteString(string(i))
		}
	}

	if m.ContainsString("abcdefghijklmnopqrstuvwxyz") {
		fmt.Println("Found the alphabet!")
	}

	if m.ContainsString("za") {
		fmt.Println("[Error] Terminal did not wrap!")
	}

	formatted := mimic.Viewer{Mimic: m, StripAnsi: true, Trim: true}
	fmt.Printf("\nFormatted View (%d columns):\n%s\n", columns, formatted.String())

	// Output:
	// Found the alphabet!
	//
	// Formatted View (26 columns):
	// abcdefghijklmnopqrstuvwxyz
	// abcdefghijklmnopqrstuvwxyz
	// abcdefghijklmnopqrstuvwxyz
}

func ExampleViewer_String() {
	m, _ := mimic.NewMimic(
		mimic.WithSize(24, 80),
		mimic.WithIdleTimeout(300*time.Millisecond),
	)

	text := "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Sed vulputate odio ut enim blandit volutpat maecenas volutpat."
	_, _ = m.WriteString(text)

	_ = m.Flush()

	formatted := mimic.Viewer{Mimic: m, StripAnsi: true, Trim: true}
	fmt.Printf("%s\n", formatted.String())

	// Output:
	// Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor i
	// ncididunt ut labore et dolore magna aliqua. Sed vulputate odio ut enim blandit v
	// olutpat maecenas volutpat.
}

func ExampleMimic_ExpectString() {
	columns := 30
	m, _ := mimic.NewMimic(
		mimic.WithSize(24, columns),
	)

	// text is Hi*16 (or, 32 letters); column width is 30
	text := strings.Repeat("Hi", 16)
	_, _ = m.WriteString(text)

	// Expect the first line (note, no newline expectations)
	if err := m.ExpectString(strings.Repeat("Hi", 15)); err == nil {
		fmt.Printf("Found: %s\n\n", strings.Repeat("Hi", 15))
	}

	// Expect the second line (note, no newline expectations)
	if err := m.ExpectString("Hi"); err != nil {
		fmt.Println("The text should have wrapped!")
	}

	_ = m.NoMoreExpectations()

	formatted := mimic.Viewer{Mimic: m, StripAnsi: true, Trim: true}
	fmt.Printf("Formatted View (%d columns):\n%s\n", columns, formatted.String())

	// Output: Found: HiHiHiHiHiHiHiHiHiHiHiHiHiHiHi
	//
	// Formatted View (30 columns):
	// HiHiHiHiHiHiHiHiHiHiHiHiHiHiHi
	// Hi
}

func ExampleMimic_ExpectString_with_ContainsString() {
	columns := 26
	m, _ := mimic.NewMimic(
		mimic.WithSize(24, columns),
		mimic.WithIdleTimeout(50*time.Millisecond),
	)

	go func() {
		// create three rows of text…
		for row := 1; row <= 3; row++ {
			for i := 'a'; i <= 'z'; i++ {
				// note we don't write \n here. Formatting defined by column width.
				_, _ = m.WriteString(string(i))
			}
		}
		_, _ = m.WriteString("\nDONE.")
	}()

	_ = m.ExpectString("DONE.")

	// force Flush and expect EOF
	// this can be omitted if you don't want to expect EOF
	m.NoMoreExpectations()

	if m.ContainsString("DONE.") {
		fmt.Printf("Found 'DONE.'\n\n")
	}

	if m.ContainsString("za") {
		fmt.Println("Terminal did not wrap!")
	}

	formatted := mimic.Viewer{Mimic: m, StripAnsi: true, Trim: true}
	fmt.Printf("Formatted View (%d columns):\n%s\n", columns, formatted.String())

	// Output:
	// Found 'DONE.'
	//
	// Formatted View (26 columns):
	// abcdefghijklmnopqrstuvwxyz
	// abcdefghijklmnopqrstuvwxyz
	// abcdefghijklmnopqrstuvwxyz
	// DONE.
}
