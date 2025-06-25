package vcs

import (
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"
	"golang.org/x/oauth2"
)

// GitHubClient is an implementation of the VCSAdapter for GitHub.
type GitHubClient struct {
	client *github.Client
}

// NewGitHubClient creates a new client for interacting with the GitHub API.
func NewGitHubClient(ctx context.Context, token string) *GitHubClient {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	return &GitHubClient{client: github.NewClient(tc)}
}

// GetPRDiff fetches the pull request diff from GitHub using the special .diff media type.
func (g *GitHubClient) GetPRDiff(ctx context.Context, owner, repo string, prNumber int) (string, error) {
	// The `github.RawOptions{Type: github.Diff}` tells the client to request
	// the text/vnd.github.v3.diff format.
	diff, _, err := g.client.PullRequests.GetRaw(ctx, owner, repo, prNumber, github.RawOptions{Type: github.Diff})
	if err != nil {
		return "", fmt.Errorf("failed to get PR diff from GitHub: %w", err)
	}
	return diff, nil
}

// PostReviewComment posts a single review comment to a pull request on GitHub.
func (g *GitHubClient) PostReviewComment(ctx context.Context, owner, repo string, prNumber int, comment *Comment) error {
	prComment := &github.PullRequestComment{
		Body: &comment.Body,
		Path: &comment.Path,
		Line: &comment.Line,
		// Using Side: "RIGHT" ensures the comment is on the new code in a side-by-side diff.
		Side: github.String("RIGHT"),
	}

	_, _, err := g.client.PullRequests.CreateComment(ctx, owner, repo, prNumber, prComment)
	if err != nil {
		return fmt.Errorf("failed to post review comment to GitHub: %w", err)
	}
	return nil
}

// PostGeneralComment posts a general comment on the PR (not tied to a specific line).
func (g *GitHubClient) PostGeneralComment(ctx context.Context, owner, repo string, prNumber int, body string) error {
	issueComment := &github.IssueComment{
		Body: &body,
	}
	_, _, err := g.client.Issues.CreateComment(ctx, owner, repo, prNumber, issueComment)
	if err != nil {
		return fmt.Errorf("failed to post general comment to GitHub: %w", err)
	}
	return nil
}
