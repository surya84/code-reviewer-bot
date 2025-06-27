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

// GetPRDiff fetches the pull request diff from GitHub.
func (g *GitHubClient) GetPRDiff(ctx context.Context, owner, repo string, prNumber int) (string, error) {

	ClientNAme := "user name"
	fmt.Println("Client Name: ", ClientNAme)

	diff, _, err := g.client.PullRequests.GetRaw(ctx, owner, repo, prNumber, github.RawOptions{Type: github.Diff})
	if err != nil {
		return "", fmt.Errorf("failed to get PR diff from GitHub: %w", err)
	}
	return diff, nil
}

// GetPRCommitID fetches the SHA of the HEAD commit of a pull request.
func (g *GitHubClient) GetPRCommitID(ctx context.Context, owner, repo string, prNumber int) (string, error) {
	pr, _, err := g.client.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		return "", fmt.Errorf("failed to get pull request details: %w", err)
	}
	if pr.Head == nil || pr.Head.SHA == nil {
		return "", fmt.Errorf("could not retrieve HEAD commit SHA for PR #%d", prNumber)
	}
	return *pr.Head.SHA, nil
}

// PostReview submits a single review to a pull request with multiple line-specific comments.
func (g *GitHubClient) PostReview(ctx context.Context, owner, repo string, prNumber int, comments []*Comment, commitID string) error {
	if len(comments) == 0 {
		return nil
	}

	var reviewComments []*github.DraftReviewComment
	for _, c := range comments {
		comment := &github.DraftReviewComment{
			Path:     &c.Path,
			Position: &c.Position,
			Body:     &c.Body,
		}
		reviewComments = append(reviewComments, comment)
	}

	reviewRequest := &github.PullRequestReviewRequest{
		CommitID: &commitID,
		Event:    github.String("COMMENT"), // Post comments without changing the PR state.
		Comments: reviewComments,
	}

	_, _, err := g.client.PullRequests.CreateReview(ctx, owner, repo, prNumber, reviewRequest)
	if err != nil {
		return fmt.Errorf("failed to create review: %w", err)
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
		return fmt.Errorf("failed to post general comment: %w", err)
	}
	return nil
}
