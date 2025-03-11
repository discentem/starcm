package diffutils

import (
	"strings"

	"github.com/google/go-cmp/cmp"
)

// GitDiff generates a Git-style line-by-line diff using go-cmp with Git diff options.
func GitDiff(text1, text2 string) string {
	// Split the input texts into lines.
	lines1 := strings.Split(text1, "\n")
	lines2 := strings.Split(text2, "\n")

	// Set the DiffOption for Git-style output.
	opts := cmp.Options{}

	// Generate the diff between the two sets of lines using go-cmp with Git style
	differences := cmp.Diff(lines1, lines2, opts...)

	var result strings.Builder

	// Process the differences output and format it in Git diff style
	for _, line := range strings.Split(differences, "\n") {
		if strings.HasPrefix(line, "-") {
			// Lines deleted from text1
			result.WriteString("- " + line[2:] + "\n")
		} else if strings.HasPrefix(line, "+") {
			// Lines added in text2
			result.WriteString("+ " + line[2:] + "\n")
		} else if strings.HasPrefix(line, " ") {
			// Unchanged lines
			result.WriteString("  " + line[2:] + "\n")
		}
	}

	return result.String()
}
