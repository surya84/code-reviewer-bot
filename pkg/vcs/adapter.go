package vcs

import "context"

// Comment represents a single review comment to be posted.
// It holds all possible location data; each client will use what it needs.
type Comment struct {
	Body     string
	Path     string
	Position int // For GitHub API: the line number within the diff hunk.
	Line     int // For Gitea API: the absolute line number in the file.
}

// VCSAdapter defines the contract for a Version Control System client.
type VCSAdapter interface {
	GetPRDiff(ctx context.Context, owner, repo string, prNumber int) (string, error)
	PostReview(ctx context.Context, owner, repo string, prNumber int, comments []*Comment, commitID string) error
	PostGeneralComment(ctx context.Context, owner, repo string, prNumber int, body string) error
	GetPRCommitID(ctx context.Context, owner, repo string, prNumber int) (string, error)
}
