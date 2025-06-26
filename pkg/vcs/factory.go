package vcs

import (
	"context"
	"fmt"

	// Corrected import path without markdown
	"github.com/surya84/code-reviewer-bot/config"
	"github.com/surya84/code-reviewer-bot/constants"
)

// NewVCSClient is a factory function that creates a VCSAdapter based on the
// provided configuration. This allows for easy extension to other VCS providers.
func NewVCSClient(ctx context.Context, cfg *config.VCSConfig) (VCSAdapter, error) {
	switch cfg.Provider {
	case constants.GITHUB:
		if cfg.GitHub.Token == "" {
			return nil, fmt.Errorf("github token is not configured")
		}
		return NewGitHubClient(ctx, cfg.GitHub.Token), nil
	case constants.GITEA: // Changed from "gitty" for clarity
		if cfg.Gitea.Token == "" {
			return nil, fmt.Errorf("gitea token is not configured")
		}
		return NewGiteaClient(ctx, cfg.Gitea.BaseURL, cfg.Gitea.Token), nil
	default:
		return nil, fmt.Errorf("unsupported VCS provider: %s", cfg.Provider)
	}
}
