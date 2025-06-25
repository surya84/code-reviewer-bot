package diffparser

import (
	"strconv"
	"strings"
)

// DiffChunk represents a single hunk of a diff for a single file.
type DiffChunk struct {
	FilePath     string
	CodeSnippet  string
	StartLineNew int // The starting line number of this hunk in the new file.
}

// Parse takes a raw diff string and splits it into analyzable hunks.
func Parse(diffStr string) []*DiffChunk {
	var chunks []*DiffChunk

	files := strings.Split(diffStr, "diff --git a/")
	for _, file := range files {
		if strings.TrimSpace(file) == "" {
			continue
		}

		parts := strings.SplitN(file, "\n", 2)
		if len(parts) < 2 {
			continue
		}

		headerParts := strings.Fields(parts[0])
		if len(headerParts) < 2 {
			continue
		}
		filePath := strings.TrimPrefix(headerParts[1], "b/")
		content := parts[1]

		// Split the content by hunk headers.
		hunks := strings.Split(content, "\n@@")
		for i, hunk := range hunks {
			if i == 0 { // Skip file mode info before the first hunk.
				continue
			}

			// Extract the starting line number for the new file.
			hunkHeaderEnd := strings.Index(hunk, "@@")
			if hunkHeaderEnd == -1 {
				continue
			}
			headerLine := hunk[:hunkHeaderEnd] // e.g., " -10,6 +10,7 "
			headerParts := strings.Fields(headerLine)
			var startLine int
			if len(headerParts) > 1 {
				// The new file info is the second part, e.g., "+10,7"
				lineInfo := strings.Split(strings.TrimPrefix(headerParts[1], "+"), ",")
				startLine, _ = strconv.Atoi(lineInfo[0])
			}

			// Re-add the separator for the snippet.
			codeSnippet := "@@" + hunk

			// Only create a chunk if it has actual changes.
			if strings.Contains(hunk, "\n+") || strings.Contains(hunk, "\n-") {
				chunk := &DiffChunk{
					FilePath:     filePath,
					CodeSnippet:  codeSnippet,
					StartLineNew: startLine,
				}
				chunks = append(chunks, chunk)
			}
		}
	}

	return chunks
}
