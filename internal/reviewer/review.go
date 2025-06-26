package reviewer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/surya84/code-reviewer-bot/config"
	"github.com/surya84/code-reviewer-bot/constants"
	"github.com/surya84/code-reviewer-bot/internal/diffparser"
	"github.com/surya84/code-reviewer-bot/pkg/vcs"
)

// PRDetails holds information about the pull request being reviewed.
type PRDetails struct {
	Owner    string
	Repo     string
	PRNumber int
}

// ReviewComment represents the structured response from the LLM.
type ReviewComment struct {
	LineContent string `json:"line_content"`
	Message     string `json:"message"`
}

// RunReview is the main function that orchestrates the entire review process.
func RunReview(ctx context.Context, g *genkit.Genkit, prDetails *PRDetails, cfg *config.Config, vcsClient vcs.VCSAdapter) (string, error) {
	log.Printf("Starting review for PR #%d in %s/%s", prDetails.PRNumber, prDetails.Owner, prDetails.Repo)

	commitID, err := vcsClient.GetPRCommitID(ctx, prDetails.Owner, prDetails.Repo, prDetails.PRNumber)
	if err != nil {
		return "", fmt.Errorf("failed to get PR commit ID: %w", err)
	}
	log.Printf("Found PR HEAD commit SHA: %s", commitID)

	diff, err := vcsClient.GetPRDiff(ctx, prDetails.Owner, prDetails.Repo, prDetails.PRNumber)
	if err != nil {
		return "", fmt.Errorf("failed to get PR diff: %w", err)
	}
	log.Println("Successfully fetched PR diff.")

	chunks := diffparser.Parse(diff)
	if len(chunks) == 0 {
		return "No reviewable changes found.", nil
	}
	log.Printf("Parsed diff into %d chunks.", len(chunks))

	var allComments []*vcs.Comment
	for _, chunk := range chunks {
		comments, err := analyzeChunk(ctx, g, cfg, chunk)
		if err != nil {
			log.Printf("Error analyzing chunk for file %s: %v", chunk.FilePath, err)
			continue
		}

		for _, llmComment := range comments {
			// Find the position of the commented line within the diff hunk.
			positionInHunk, err := findPositionForLineContent(chunk, llmComment.LineContent)
			if err != nil {
				log.Printf("Could not find position for line content in file %s: %v", chunk.FilePath, err)
				continue
			}
			allComments = append(allComments, &vcs.Comment{
				Body:     llmComment.Message,
				Path:     chunk.FilePath,
				Position: positionInHunk,
			})
		}
	}

	if len(allComments) > 0 {
		log.Printf("Submitting a review with %d comments.", len(allComments))
		err := vcsClient.PostReview(ctx, prDetails.Owner, prDetails.Repo, prDetails.PRNumber, allComments, commitID)
		if err != nil {
			return "", fmt.Errorf("failed to post review: %w", err)
		}
	} else {
		vcsClient.PostGeneralComment(ctx, prDetails.Owner, prDetails.Repo, prDetails.PRNumber, "✅ AI Review Complete: No issues found. Great work!")
	}

	resultMessage := fmt.Sprintf("Review complete. Submitted %d comments.", len(allComments))
	log.Println(resultMessage)
	return resultMessage, nil
}

// analyzeChunk sends a single diff chunk to the LLM.
func analyzeChunk(ctx context.Context, g *genkit.Genkit, cfg *config.Config, chunk *diffparser.DiffChunk) ([]ReviewComment, error) {
	prompt, err := preparePrompt(cfg.ReviewPrompt, chunk.FilePath, chunk.CodeSnippet)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare prompt: %w", err)
	}

	res, err := genkit.Generate(ctx, g, ai.WithModelName(cfg.LLM.ModelName), ai.WithPrompt(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate LLM response: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to generate LLM response: %w", err)
	}

	responseText := res.Text()
	if responseText == "" {
		return nil, fmt.Errorf("failed to get text from LLM response")
	}

	startIndex := strings.Index(responseText, "[")
	endIndex := strings.LastIndex(responseText, "]")
	if startIndex == -1 || endIndex == -1 || endIndex < startIndex {
		return nil, nil // No valid JSON found
	}
	jsonString := responseText[startIndex : endIndex+1]

	// Sanitize the JSON string to remove invalid characters like tabs before parsing.
	sanitizedJSON := strings.ReplaceAll(jsonString, "\t", " ")

	var comments []ReviewComment
	if err := json.Unmarshal([]byte(sanitizedJSON), &comments); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON response: %w", err)
	}
	return comments, nil
}

// findPositionForLineContent iterates through a diff hunk to find the 1-based position of a specific line.
func findPositionForLineContent(chunk *diffparser.DiffChunk, lineContent string) (int, error) {
	// A more robust normalization function that replaces all sequences of whitespace
	// (spaces, tabs, etc.) with a single space for a more reliable comparison.
	normalize := func(s string) string {
		return strings.Join(strings.Fields(s), " ")
	}

	// Normalize the target content from the LLM.
	target := normalize(lineContent)

	lines := strings.Split(chunk.CodeSnippet, "\n")
	for i, line := range lines {
		// Normalize the current line from the diff for comparison.
		current := normalize(line)
		if current == target {
			// The position is the 1-based index of the line in the hunk.
			return i + 1, nil
		}
	}
	return -1, fmt.Errorf("line content not found in diff hunk: '%s'", lineContent)
}

// preparePrompt populates the Go template for the LLM prompt.
func preparePrompt(promptTmpl, filePath, codeSnippet string) (string, error) {
	tmpl, err := template.New(constants.REVIEW_PROMPT).Parse(promptTmpl)
	if err != nil {
		return "", err
	}
	data := struct {
		FilePath    string
		CodeSnippet string
	}{
		FilePath:    filePath,
		CodeSnippet: codeSnippet,
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
