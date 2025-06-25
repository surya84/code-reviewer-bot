package diffparser

import (
	"strings"
)

// DiffChunk represents a piece of a diff for a single file, suitable for analysis.
type DiffChunk struct {
	FilePath    string
	CodeSnippet string
}

// Parse takes a raw diff string (in unified format) and splits it into
// chunks that can be analyzed independently.
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
		
		// Extract file path from the first line (e.g., "path/to/file.go b/path/to/file.go")
		headerParts := strings.Fields(parts[0])
		if len(headerParts) < 2 {
			continue
		}
		filePath := strings.TrimPrefix(headerParts[1], "b/")

		// The rest is the content of the diff for this file
		content := parts[1]
		
		hunks := strings.Split(content, "\n@@")
		for i, hunk := range hunks {
			if i == 0 { // First part is usually file mode info, not a hunk
				continue
			}
			
			// Re-add the separator for the snippet
			codeSnippet := "@@" + hunk
			
			// We only care about hunks that have actual changes
			if strings.Contains(hunk, "\n+") || strings.Contains(hunk, "\n-") {
				chunk := &DiffChunk{
					FilePath:    filePath,
					CodeSnippet: codeSnippet,
				}
				chunks = append(chunks, chunk)
			}
		}
	}

	return chunks
}
