package vcs

import "context"

// Comment represents a single review comment to be posted.
type Comment struct {
	Body     string
	Path     string
	Line     int
}

// VCSAdapter is the interface that defines the contract for a Version Control System client.
// This allows the application to support different providers (like GitHub, GitLab, etc.)
// by implementing this interface. This is the core of the Adapter design pattern.
type VCSAdapter interface {
	// GetPRDiff fetches the diff of a pull request as a raw string.
	GetPRDiff(ctx context.Context, owner, repo string, prNumber int) (string, error)

	// PostReviewComment posts a single line-specific comment to a pull request.
	PostReviewComment(ctx context.Context, owner, repo string, prNumber int, comment *Comment) error
	
	// PostGeneralComment posts a comment not tied to a specific line of code,
	// useful for summaries or error messages.
	PostGeneralComment(ctx context.Context, owner, repo string, prNumber int, body string) error
}
