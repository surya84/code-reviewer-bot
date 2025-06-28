package vcs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// GiteaClient implements the VCSAdapter for Gitea.
type GiteaClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewGiteaClient creates a new client for interacting with the Gitea API.
func NewGiteaClient(ctx context.Context, baseURL, token string) *GiteaClient {
	return &GiteaClient{
		BaseURL:    baseURL,
		Token:      token,
		HTTPClient: &http.Client{},
	}
}

func (g *GiteaClient) newRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+g.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

// GetPRDiff fetches a Pull Request's diff from Gitea.
func (g *GiteaClient) GetPRDiff(ctx context.Context, owner, repo string, prIndex int) (string, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/pulls/%d.diff", g.BaseURL, owner, repo, prIndex)
	req, err := g.newRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	resp, err := g.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gitea API returned status %d for GetPRDiff", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	return string(body), err
}

// GetPRCommitID is not used by this Gitea client strategy, but is required by the interface.
func (g *GiteaClient) GetPRCommitID(ctx context.Context, owner, repo string, prIndex int) (string, error) {
	return "", nil
}

// PostReview for Gitea now iterates and posts each comment individually to the main conversation thread.
func (g *GiteaClient) PostReview(ctx context.Context, owner, repo string, prIndex int, comments []*Comment, commitID string) error {
	if len(comments) == 0 {
		return nil
	}

	for _, comment := range comments {
		// Format the comment body to include all necessary context (file and line).
		// This creates a clear, readable comment in the "Conversation" tab.
		formattedBody := fmt.Sprintf("**Review for `%s` (Line %d):**\n\n> %s", comment.Path, comment.Line, comment.Body)
		if err := g.PostGeneralComment(ctx, owner, repo, prIndex, formattedBody); err != nil {
			// Log the error but continue trying to post other comments.
			log.Printf("Failed to post general comment to Gitea PR #%d: %v", prIndex, err)
		}
	}
	return nil
}

// PostGeneralComment posts a general comment to the PR's issue thread.
// This is the correct and stable endpoint for posting to the "Conversation" tab.
func (g *GiteaClient) PostGeneralComment(ctx context.Context, owner, repo string, prIndex int, body string) error {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", g.BaseURL, owner, repo, prIndex)
	payload := map[string]string{"body": body}
	jsonBody, _ := json.Marshal(payload)
	req, err := g.newRequest(ctx, "POST", endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	resp, err := g.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Gitea API returned status %d for PostGeneralComment: %s", resp.StatusCode, string(respBody))
	}
	log.Printf("Successfully posted general comment to Gitea PR #%d", prIndex)
	return nil
}
