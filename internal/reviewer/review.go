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
	"github.com/surya84/code-reviewer-bot/internal/diffparser"
	"github.com/surya84/code-reviewer-bot/pkg/vcs"
)

// FlowInput is no longer needed as we are not using Genkit flows.

// PRDetails holds information about the pull request being reviewed.
type PRDetails struct {
	Owner    string
	Repo     string
	PRNumber int
}

// ReviewComment represents the structured response from the LLM.
type ReviewComment struct {
	LineNumber int    `json:"lineNumber"`
	Comment    string `json:"comment"`
}

// RunReview is the main function that orchestrates the entire review process.
// It is a standard Go function, not a Genkit Flow.
func RunReview(ctx context.Context, g *genkit.Genkit, prDetails *PRDetails, cfg *config.Config, vcsClient vcs.VCSAdapter) (string, error) {
	log.Printf("Starting review for PR #%d in %s/%s", prDetails.PRNumber, prDetails.Owner, prDetails.Repo)

	// 1. Fetch the PR diff.
	diff, err := vcsClient.GetPRDiff(ctx, prDetails.Owner, prDetails.Repo, prDetails.PRNumber)
	if err != nil {
		return "", fmt.Errorf("failed to get PR diff: %w", err)
	}
	log.Println("Successfully fetched PR diff.")

	// 2. Parse the diff.
	chunks := diffparser.Parse(diff)
	if len(chunks) == 0 {
		log.Println("No reviewable changes found in diff.")
		return "No reviewable changes found.", nil
	}
	log.Printf("Parsed diff into %d chunks.", len(chunks))

	// 3. Process each chunk.
	totalComments := 0
	for _, chunk := range chunks {
		comments, err := analyzeChunk(ctx, g, cfg, chunk)
		if err != nil {
			log.Printf("Error analyzing chunk for file %s: %v", chunk.FilePath, err)
			continue
		}

		if len(comments) > 0 {
			log.Printf("Found %d comments for file %s", len(comments), chunk.FilePath)
			for _, llmComment := range comments {
				// 4. Post comments back to VCS.
				comment := &vcs.Comment{
					Body: llmComment.Comment,
					Path: chunk.FilePath,
					Line: llmComment.LineNumber,
				}
				err := vcsClient.PostReviewComment(ctx, prDetails.Owner, prDetails.Repo, prDetails.PRNumber, comment)
				if err != nil {
					log.Printf("Failed to post comment to %s at line %d: %v", chunk.FilePath, llmComment.LineNumber, err)
				} else {
					log.Printf("Posted comment to %s at line %d", chunk.FilePath, llmComment.LineNumber)
					totalComments++
				}
			}
		}
	}

	resultMessage := fmt.Sprintf("Review complete. Posted %d comments.", totalComments)
	log.Println(resultMessage)

	if totalComments == 0 {
		vcsClient.PostGeneralComment(ctx, prDetails.Owner, prDetails.Repo, prDetails.PRNumber, "✅ AI Review Complete: No issues found. Great work!")
	}

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

	jsonString := strings.TrimSpace(responseText)
	jsonString = strings.TrimPrefix(jsonString, "```json")
	jsonString = strings.TrimPrefix(jsonString, "```")

	if jsonString == "" {
		log.Println("LLM returned an empty JSON string. No comments to parse.")
		return nil, nil
	}

	var comments []ReviewComment
	if err := json.Unmarshal([]byte(jsonString), &comments); err != nil {
		log.Printf("Failed to unmarshal LLM response into JSON. Raw response: '%s'", jsonString)
		return nil, fmt.Errorf("failed to parse LLM JSON response: %w", err)
	}

	return comments, nil
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
