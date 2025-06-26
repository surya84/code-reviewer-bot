package webhook

import (
	"context"
	"log"
	"net/http"

	"github.com/firebase/genkit/go/genkit"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v62/github"
	"github.com/surya84/code-reviewer-bot/config"
	"github.com/surya84/code-reviewer-bot/constants"
	"github.com/surya84/code-reviewer-bot/internal/reviewer"
	"github.com/surya84/code-reviewer-bot/pkg/vcs"
)

// GitHubWebhookHandler no longer stores a reference to a flow.
type GitHubWebhookHandler struct {
	g      *genkit.Genkit
	config *config.Config
	secret []byte
}

// NewGitHubWebhookHandler is simplified.
func NewGitHubWebhookHandler(g *genkit.Genkit, cfg *config.Config, secret string) (*GitHubWebhookHandler, error) {
	return &GitHubWebhookHandler{
		g:      g,
		config: cfg,
		secret: []byte(secret),
	}, nil
}

// Handle is the Gin handler function for the webhook endpoint.
func (h *GitHubWebhookHandler) Handle(c *gin.Context) {
	payload, err := github.ValidatePayload(c.Request, h.secret)
	if err != nil {
		log.Printf("Error validating webhook payload: %v", err)
		c.String(http.StatusForbidden, "Forbidden")
		return
	}
	event, err := github.ParseWebHook(github.WebHookType(c.Request), payload)
	if err != nil {
		log.Printf("Error parsing webhook event: %v", err)
		c.String(http.StatusBadRequest, "Bad Request")
		return
	}
	switch event := event.(type) {
	case *github.PullRequestEvent:
		action := event.GetAction()
		if action == constants.OPENED || action == constants.SYNCHRONIZE {
			log.Printf("Received GitHub PR event: %s for PR #%d", action, event.GetNumber())
			go h.processPullRequest(event)
		} else {
			log.Printf("Ignoring GitHub PR action: %s", action)
		}
		c.String(http.StatusOK, "Event received.")
	default:
		log.Printf("Ignoring GitHub webhook event type: %T", event)
		c.String(http.StatusOK, "Event type ignored.")
	}
}

// processPullRequest now explicitly creates a GitHubClient.
func (h *GitHubWebhookHandler) processPullRequest(event *github.PullRequestEvent) {
	ctx := context.Background()

	pr := event.GetPullRequest()
	if pr.GetState() != constants.OPEN {
		log.Printf("Ignoring PR #%d because its state is '%s'", pr.GetNumber(), pr.GetState())
		return
	}
	prDetails := &reviewer.PRDetails{
		Owner:    pr.Base.Repo.GetOwner().GetLogin(),
		Repo:     pr.Base.Repo.GetName(),
		PRNumber: pr.GetNumber(),
	}

	// CORRECTED: Explicitly create a GitHub client, ignoring the static config provider.
	vcsClient := vcs.NewGitHubClient(ctx, h.config.VCS.GitHub.Token)

	_, err := reviewer.RunReview(ctx, h.g, prDetails, h.config, vcsClient)
	if err != nil {
		log.Printf("Code review failed for GitHub PR #%d: %v", prDetails.PRNumber, err)
		vcsClient.PostGeneralComment(ctx, prDetails.Owner, prDetails.Repo, prDetails.PRNumber, "❌ AI Review Failed: An internal error occurred.")
	}
}
