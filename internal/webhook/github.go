package webhook

import (
	"context"
	"log"
	"net/http"

	"github.com/firebase/genkit/go/genkit"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v62/github"
	"github.com/surya84/code-reviewer-bot/config"
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
		if action == "opened" || action == "synchronize" {
			log.Printf("Received pull_request '%s' event for PR #%d", action, event.GetNumber())
			go h.processPullRequest(c.Request.Context(), event)
			c.String(http.StatusOK, "Event received and is being processed.")
		} else {
			log.Printf("Ignoring pull_request action: %s", action)
			c.String(http.StatusOK, "Event ignored.")
		}
	default:
		log.Printf("Ignoring webhook event type: %T", event)
		c.String(http.StatusOK, "Event type ignored.")
	}
}

// processPullRequest calls the main review function directly.
func (h *GitHubWebhookHandler) processPullRequest(ctx context.Context, event *github.PullRequestEvent) {
	pr := event.GetPullRequest()
	if pr.GetState() != "open" {
		log.Printf("Ignoring PR #%d because its state is '%s'", pr.GetNumber(), pr.GetState())
		return
	}
	prDetails := &reviewer.PRDetails{
		Owner:    pr.Base.Repo.GetOwner().GetLogin(),
		Repo:     pr.Base.Repo.GetName(),
		PRNumber: pr.GetNumber(),
	}
	vcsClient, err := vcs.NewVCSClient(ctx, &h.config.VCS)
	if err != nil {
		log.Printf("Error creating VCS client for PR #%d: %v", prDetails.PRNumber, err)
		return
	}
	
	// Directly call the standard Go function that orchestrates the review.
	_, err = reviewer.RunReview(ctx, h.g, prDetails, h.config, vcsClient)
	if err != nil {
		log.Printf("Code review flow failed for PR #%d: %v", prDetails.PRNumber, err)
		vcsClient.PostGeneralComment(ctx, prDetails.Owner, prDetails.Repo, prDetails.PRNumber, "❌ AI Review Failed: An internal error occurred.")
	}
}
