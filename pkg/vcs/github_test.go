package vcs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v62/github"
	"github.com/stretchr/testify/assert"
)

// setupGitHubTestServer creates a mock HTTP server and a GitHubClient pointed to it.
func setupGitHubTestServer(t *testing.T, handler http.HandlerFunc) (*GitHubClient, *httptest.Server) {
	server := httptest.NewServer(handler)

	// Create a client that points to our mock server
	client, err := github.NewClient(server.Client()).WithEnterpriseURLs(server.URL, server.URL)
	assert.NoError(t, err)

	return &GitHubClient{client: client}, server
}

func TestGitHubClient_GetPRDiff(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		expectedDiff := "diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n@@ -1,1 +1,1 @@\n-hello\n+world"
		handler := func(w http.ResponseWriter, r *http.Request) {
			// CORRECTED: The client prepends /api/v3/ to the path.
			assert.Equal(t, "/api/v3/repos/owner/repo/pulls/1", r.URL.Path)
			assert.Equal(t, "application/vnd.github.v3.diff", r.Header.Get("Accept"))
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, expectedDiff)
		}
		client, server := setupGitHubTestServer(t, handler)
		defer server.Close()

		diff, err := client.GetPRDiff(context.Background(), "owner", "repo", 1)
		assert.NoError(t, err)
		assert.Equal(t, expectedDiff, diff)
	})

	t.Run("Failure - Not Found", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}
		client, server := setupGitHubTestServer(t, handler)
		defer server.Close()

		_, err := client.GetPRDiff(context.Background(), "owner", "repo", 1)
		assert.Error(t, err)
	})
}

func TestGitHubClient_GetPRCommitID(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		expectedSHA := "abcdef1234567890"
		handler := func(w http.ResponseWriter, r *http.Request) {
			// CORRECTED: The client prepends /api/v3/ to the path.
			assert.Equal(t, "/api/v3/repos/owner/repo/pulls/1", r.URL.Path)
			resp := github.PullRequest{
				Head: &github.PullRequestBranch{SHA: &expectedSHA},
			}
			json.NewEncoder(w).Encode(resp)
		}
		client, server := setupGitHubTestServer(t, handler)
		defer server.Close()

		sha, err := client.GetPRCommitID(context.Background(), "owner", "repo", 1)
		assert.NoError(t, err)
		assert.Equal(t, expectedSHA, sha)
	})
}

func TestGitHubClient_PostReview(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			// CORRECTED: The client prepends /api/v3/ to the path.
			assert.Equal(t, "/api/v3/repos/owner/repo/pulls/1/reviews", r.URL.Path)

			body, _ := io.ReadAll(r.Body)
			var reviewReq github.PullRequestReviewRequest
			json.Unmarshal(body, &reviewReq)

			assert.Equal(t, "test-commit-id", *reviewReq.CommitID)
			assert.Len(t, reviewReq.Comments, 1)
			assert.Equal(t, "This is a test comment", *reviewReq.Comments[0].Body)

			w.WriteHeader(http.StatusCreated)
		}
		client, server := setupGitHubTestServer(t, handler)
		defer server.Close()

		comments := []*Comment{{Body: "This is a test comment", Path: "main.go", Position: 5}}
		err := client.PostReview(context.Background(), "owner", "repo", 1, comments, "test-commit-id")
		assert.NoError(t, err)
	})

	t.Run("Failure - API Error", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnprocessableEntity)
		}
		client, server := setupGitHubTestServer(t, handler)
		defer server.Close()

		comments := []*Comment{{Body: "This comment will fail", Path: "main.go", Position: 5}}
		err := client.PostReview(context.Background(), "owner", "repo", 1, comments, "test-commit-id")
		assert.Error(t, err)
	})
}

func TestGitHubClient_PostGeneralComment(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			// CORRECTED: The client prepends /api/v3/ to the path.
			assert.Equal(t, "/api/v3/repos/owner/repo/issues/1/comments", r.URL.Path)
			w.WriteHeader(http.StatusCreated)
		}
		client, server := setupGitHubTestServer(t, handler)
		defer server.Close()

		err := client.PostGeneralComment(context.Background(), "owner", "repo", 1, "Summary comment")
		assert.NoError(t, err)
	})
}
