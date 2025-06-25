package vcs

import (
	"context"
	"fmt"
	
	// Corrected import path without markdown
	"github.com/surya84/code-reviewer-bot/config"
)

// NewVCSClient is a factory function that creates a VCSAdapter based on the
// provided configuration. This allows for easy extension to other VCS providers.
func NewVCSClient(ctx context.Context, cfg *config.VCSConfig) (VCSAdapter, error) {
	switch cfg.Provider {
	case "github":
		if cfg.GitHub.Token == "" {
			return nil, fmt.Errorf("github token is not configured")
		}
		return NewGitHubClient(ctx, cfg.GitHub.Token), nil
	case "gitty":
		if cfg.Gitty.Token == "" {
			return nil, fmt.Errorf("gitty token is not configured")
		}
		// This returns the placeholder Gitty client.
		return NewGittyClient(ctx, cfg.Gitty.BaseURL, cfg.Gitty.Token), nil
	default:
		return nil, fmt.Errorf("unsupported VCS provider: %s", cfg.Provider)
	}
}
