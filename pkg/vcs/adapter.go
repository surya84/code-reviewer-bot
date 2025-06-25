package vcs

import "context"

// Comment represents a single review comment to be posted.
type Comment struct {
	Body     string
	Path     string
	Position int // The 1-based line index within the diff hunk.
}

// VCSAdapter defines the contract for a Version Control System client.
type VCSAdapter interface {
	GetPRDiff(ctx context.Context, owner, repo string, prNumber int) (string, error)
	// PostReview submits a single review with multiple comments.
	PostReview(ctx context.Context, owner, repo string, prNumber int, comments []*Comment, commitID string) error
	PostGeneralComment(ctx context.Context, owner, repo string, prNumber int, body string) error
	GetPRCommitID(ctx context.Context, owner, repo string, prNumber int) (string, error)
}
