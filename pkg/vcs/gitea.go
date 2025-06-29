package vcs

import (
	"context"
	"fmt"
	"log"
	"strings"

	"code.gitea.io/sdk/gitea"
)

// GiteaClient implements the VCSAdapter for Gitea using the official Go SDK.
type GiteaClient struct {
	client *gitea.Client
}

// NewGiteaClient creates a new client for interacting with the Gitea API.
func NewGiteaClient(ctx context.Context, baseURL, token string) *GiteaClient {
	c, err := gitea.NewClient(baseURL, gitea.SetToken(token))
	if err != nil {
		log.Fatalf("Failed to create Gitea client: %v", err)
	}
	return &GiteaClient{client: c}
}

// GetPRDiff fetches a Pull Request's diff from Gitea.
func (g *GiteaClient) GetPRDiff(ctx context.Context, owner, repo string, prIndex int) (string, error) {
	diff, _, err := g.client.GetPullRequestDiff(owner, repo, int64(prIndex), gitea.PullRequestDiffOptions{})
	if err != nil {
		return "", fmt.Errorf("Gitea SDK failed to get PR diff: %w", err)
	}
	return string(diff), nil
}

// GetPRCommitID fetches the SHA of the HEAD commit of a Gitea Pull Request.
func (g *GiteaClient) GetPRCommitID(ctx context.Context, owner, repo string, prIndex int) (string, error) {
	pr, _, err := g.client.GetPullRequest(owner, repo, int64(prIndex))
	if err != nil {
		return "", fmt.Errorf("Gitea SDK failed to get PR details: %w", err)
	}
	if pr.Head == nil || pr.Head.Sha == "" {
		return "", fmt.Errorf("Gitea API response did not contain HEAD commit SHA")
	}
	return pr.Head.Sha, nil
}

// PostReview submits a single review to a Gitea pull request with multiple line-specific comments.
// This is the definitive, robust method.
func (g *GiteaClient) PostReview(ctx context.Context, owner, repo string, prIndex int, comments []*Comment, commitID string) error {
	if len(comments) == 0 {
		return nil // Nothing to do.
	}

	// The SDK's CreatePullReview function takes a list of gitea.CreatePullReviewComment.
	var giteaComments []gitea.CreatePullReviewComment
	for _, c := range comments {
		// CORRECTED: The Gitea API's review system requires the absolute line number in the new file,
		// which is correctly calculated and passed in the `c.Line` field.
		giteaComments = append(giteaComments, gitea.CreatePullReviewComment{
			Path:       c.Path,
			Body:       c.Body,
			NewLineNum: int64(c.Line),
		})
	}

	// Create the review options payload.
	opts := gitea.CreatePullReviewOptions{
		State:    gitea.ReviewStateComment, // Post comments without changing the PR state.
		CommitID: commitID,
		Comments: giteaComments,
	}

	// Call the SDK function to create the review.
	_, _, err := g.client.CreatePullReview(owner, repo, int64(prIndex), opts)
	if err != nil {
		// Fallback for older Gitea instances that might not support batch reviews.
		if strings.Contains(err.Error(), "404 Not Found") {
			log.Println("WARNING: Gitea instance may be too old to support batch reviews. Falling back to posting a single summary comment.")
			var summary strings.Builder
			summary.WriteString("### AI Code Review Summary\n\nI was unable to post inline comments as this Gitea version might not support it. Here is a summary of the feedback:\n\n")
			for _, c := range comments {
				summary.WriteString(fmt.Sprintf("- **File `%s` (Line %d):** %s\n", c.Path, c.Line, c.Body))
			}
			return g.PostGeneralComment(ctx, owner, repo, prIndex, summary.String())
		}
		return fmt.Errorf("Gitea SDK failed to create review: %w", err)
	}

	log.Printf("Successfully submitted review to Gitea PR #%d", prIndex)
	return nil
}

// PostGeneralComment posts a general comment to the PR's issue thread.
func (g *GiteaClient) PostGeneralComment(ctx context.Context, owner, repo string, prIndex int, body string) error {
	opts := gitea.CreateIssueCommentOption{Body: body}
	_, _, err := g.client.CreateIssueComment(owner, repo, int64(prIndex), opts)
	if err != nil {
		return fmt.Errorf("Gitea SDK failed to post general comment: %w", err)
	}
	return nil
}
