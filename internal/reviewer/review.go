package reviewer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"regexp"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/surya84/code-reviewer-bot/config"
	"github.com/surya84/code-reviewer-bot/internal/diffparser"
	"github.com/surya84/code-reviewer-bot/pkg/vcs"
)

// PRDetails holds information about the pull request being reviewed.
type PRDetails struct {
	Owner    string
	Repo     string
	PRNumber int
}

// ReviewComment represents the structured response from the LLM, based on the line_content prompt.
type ReviewComment struct {
	LineContent string `json:"line_content"`
	Message     string `json:"message"`
}

// RunReview is the main function that orchestrates the entire review process.
func RunReview(ctx context.Context, g *genkit.Genkit, prDetails *PRDetails, cfg *config.Config, vcsClient vcs.VCSAdapter) (string, error) {
	log.Printf("Starting review for PR #%d in %s/%s", prDetails.PRNumber, prDetails.Owner, prDetails.Repo)

	commitID, err := vcsClient.GetPRCommitID(ctx, prDetails.Owner, prDetails.Repo, prDetails.PRNumber)
	if err != nil {
		log.Printf("Warning: could not get PR commit ID: %v", err)
	} else {
		log.Printf("Found PR HEAD commit SHA: %s", commitID)
	}

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
			// Find both the position-in-hunk and the absolute file line number for the commented line.
			positionInHunk, fileLineNumber, err := findLocationForLineContent(chunk, llmComment.LineContent)
			if err != nil {
				log.Printf("Could not find location for line content in file %s: %v", chunk.FilePath, err)
				continue
			}
			// Create a comment object with all necessary information for any VCS.
			allComments = append(allComments, &vcs.Comment{
				Body:     llmComment.Message,
				Path:     chunk.FilePath,
				Position: positionInHunk, // For GitHub
				Line:     fileLineNumber, // For Gitea
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
		log.Println("No comments to post. Submitting a general comment.")
		vcsClient.PostGeneralComment(ctx, prDetails.Owner, prDetails.Repo, prDetails.PRNumber, "✅ AI Review Complete: No issues found. Great work!")
	}

	resultMessage := fmt.Sprintf("Review complete. Submitted %d comments.", len(allComments))
	log.Println(resultMessage)
	return resultMessage, nil
}

// sanitizeJSONString removes common JSON formatting errors from LLM responses.
func sanitizeJSONString(s string) string {
	startIndex := strings.Index(s, "[")
	endIndex := strings.LastIndex(s, "]")
	if startIndex == -1 || endIndex == -1 || endIndex < startIndex {
		return ""
	}
	s = s[startIndex : endIndex+1]
	replacer := strings.NewReplacer("\n", " ", "\t", " ", "\r", " ")
	s = replacer.Replace(s)
	re := regexp.MustCompile(`,(\s*[\}\]])`)
	return re.ReplaceAllString(s, "$1")
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

	sanitizedJSON := sanitizeJSONString(responseText)
	if sanitizedJSON == "" {
		log.Printf("Could not find valid JSON array in LLM response. Raw response: '%s'", responseText)
		return nil, nil
	}

	var comments []ReviewComment
	if err := json.Unmarshal([]byte(sanitizedJSON), &comments); err != nil {
		log.Printf("Failed to unmarshal sanitized JSON. Sanitized: '%s', Error: %v", sanitizedJSON, err)
		return nil, fmt.Errorf("failed to parse LLM JSON response: %w", err)
	}
	return comments, nil
}

// findLocationForLineContent iterates through a diff hunk to find both the 1-based position
// and the absolute file line number for a specific line of code.
func findLocationForLineContent(chunk *diffparser.DiffChunk, lineContent string) (int, int, error) {
	normalize := func(s string) string {
		return strings.Join(strings.Fields(s), " ")
	}

	target := normalize(lineContent)
	if target == "" {
		return -1, -1, fmt.Errorf("LLM provided empty line content")
	}

	lines := strings.Split(chunk.CodeSnippet, "\n")
	hunkPosition := 0
	currentFileNumber := chunk.StartLineNew

	for i, line := range lines {
		hunkPosition = i + 1 // 1-based position within the hunk

		if strings.HasPrefix(line, "@@") || strings.HasPrefix(line, "-") {
			continue
		}

		// Normalize the current line from the diff for comparison.
		current := normalize(line)
		if current == target {
			// Ensure the matched line is an added line, which is what we should be commenting on.
			if !strings.HasPrefix(strings.TrimSpace(line), "+") {
				return -1, -1, fmt.Errorf("matched line is not an added line ('+'): '%s'", line)
			}
			// Return both the position in the hunk and the absolute file line number.
			return hunkPosition, currentFileNumber, nil
		}

		// Increment the file line counter for both context (' ') and added ('+') lines.
		currentFileNumber++
	}
	return -1, -1, fmt.Errorf("line content not found in diff hunk: '%s'", lineContent)
}

// preparePrompt populates the Go template for the LLM prompt.
func preparePrompt(promptTmpl, filePath, codeSnippet string) (string, error) {
	tmpl, err := template.New("review_prompt").Parse(promptTmpl)
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
