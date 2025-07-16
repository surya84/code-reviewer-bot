package vcs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"code.gitea.io/sdk/gitea"
	"github.com/stretchr/testify/assert"
)

// setupGiteaTestServer now creates a more robust mock server using a ServeMux
// to handle multiple endpoints, including the initial version check.
func setupGiteaTestServer(t *testing.T) (*GiteaClient, *http.ServeMux, *httptest.Server) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	// Default handler for the version check the SDK always performs.
	mux.HandleFunc("/api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"version":"1.21.0"}`)
	})

	client, err := gitea.NewClient(server.URL, gitea.SetToken("test-token"))
	assert.NoError(t, err)

	return &GiteaClient{client: client}, mux, server
}

func TestGiteaClient_GetPRDiff(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		client, mux, server := setupGiteaTestServer(t)
		defer server.Close()

		expectedDiff := "diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go"
		// Register the specific handler for this test case.
		mux.HandleFunc("/api/v1/repos/owner/repo/pulls/1.diff", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, expectedDiff)
		})

		diff, err := client.GetPRDiff(context.Background(), "owner", "repo", 1)
		assert.NoError(t, err)
		assert.Equal(t, expectedDiff, diff)
	})

	t.Run("Failure - Not Found", func(t *testing.T) {
		client, mux, server := setupGiteaTestServer(t)
		defer server.Close()

		mux.HandleFunc("/api/v1/repos/owner/repo/pulls/2.diff", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		_, err := client.GetPRDiff(context.Background(), "owner", "repo", 2)
		assert.Error(t, err)
	})
}

func TestGiteaClient_GetPRCommitID(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		client, mux, server := setupGiteaTestServer(t)
		defer server.Close()

		expectedSHA := "abcdef1234567890"
		mux.HandleFunc("/api/v1/repos/owner/repo/pulls/1", func(w http.ResponseWriter, r *http.Request) {
			resp := gitea.PullRequest{
				Head: &gitea.PRBranchInfo{Sha: expectedSHA},
			}
			json.NewEncoder(w).Encode(resp)
		})

		sha, err := client.GetPRCommitID(context.Background(), "owner", "repo", 1)
		assert.NoError(t, err)
		assert.Equal(t, expectedSHA, sha)
	})
}

func TestGiteaClient_PostReview(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		client, mux, server := setupGiteaTestServer(t)
		defer server.Close()

		mux.HandleFunc("/api/v1/repos/owner/repo/pulls/1/reviews", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)

			body, _ := io.ReadAll(r.Body)
			var reviewReq gitea.CreatePullReviewOptions
			json.Unmarshal(body, &reviewReq)

			assert.Equal(t, "test-commit-id", reviewReq.CommitID)
			assert.Len(t, reviewReq.Comments, 1)
			assert.Equal(t, "Gitea test comment", reviewReq.Comments[0].Body)

			w.WriteHeader(http.StatusCreated)
			// CORRECTED: Return a minimal valid JSON response.
			fmt.Fprint(w, `{}`)
		})

		comments := []*Comment{{Body: "Gitea test comment", Path: "main.go", Position: 10}}
		err := client.PostReview(context.Background(), "owner", "repo", 1, comments, "test-commit-id")
		assert.NoError(t, err)
	})
}

func TestGiteaClient_PostGeneralComment(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		client, mux, server := setupGiteaTestServer(t)
		defer server.Close()

		mux.HandleFunc("/api/v1/repos/owner/repo/issues/1/comments", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusCreated)
			// CORRECTED: Return a minimal valid JSON response.
			fmt.Fprint(w, `{}`)
		})

		err := client.PostGeneralComment(context.Background(), "owner", "repo", 1, "Gitea summary")
		assert.NoError(t, err)
	})
}
