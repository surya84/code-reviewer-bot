package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"

	"github.com/firebase/genkit/go/genkit"
	"github.com/gin-gonic/gin"
	"github.com/surya84/code-reviewer-bot/config"
	"github.com/surya84/code-reviewer-bot/constants"
	"github.com/surya84/code-reviewer-bot/internal/reviewer"
	"github.com/surya84/code-reviewer-bot/pkg/vcs"
)

// GiteaPullRequestHook represents the structure of Gitea's PR webhook payload.
type GiteaPullRequestHook struct {
	Secret      string `json:"secret"`
	Action      string `json:"action"`
	Number      int64  `json:"number"`
	PullRequest struct {
		State string `json:"state"`
	} `json:"pull_request"`
	Repository struct {
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
		Name string `json:"name"`
	} `json:"repository"`
}

type GiteaWebhookHandler struct {
	g      *genkit.Genkit
	config *config.Config
	secret string
}

func NewGiteaWebhookHandler(g *genkit.Genkit, cfg *config.Config, secret string) (*GiteaWebhookHandler, error) {
	return &GiteaWebhookHandler{g: g, config: cfg, secret: secret}, nil
}

func (h *GiteaWebhookHandler) Handle(c *gin.Context) {
	signature := c.GetHeader("X-Gitea-Signature")
	body, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if signature != expectedSignature {
		c.String(http.StatusForbidden, "Forbidden: Invalid signature")
		return
	}

	var payload GiteaPullRequestHook
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.String(http.StatusBadRequest, "Bad Request")
		return
	}

	action := payload.Action
	if action == constants.OPENED || action == constants.SYNCHRONIZE {
		log.Printf("Received Gitea PR event: %s for PR #%d", action, payload.Number)
		go h.processPullRequest(&payload)
		c.String(http.StatusOK, "Event received.")
	} else {
		log.Printf("Ignoring Gitea PR action: %s", action)
		c.String(http.StatusOK, "Event ignored.")
	}
}

func (h *GiteaWebhookHandler) processPullRequest(payload *GiteaPullRequestHook) {
	ctx := context.Background()
	prDetails := &reviewer.PRDetails{
		Owner:    payload.Repository.Owner.Login,
		Repo:     payload.Repository.Name,
		PRNumber: int(payload.Number),
	}

	vcsClient := vcs.NewGiteaClient(ctx, h.config.VCS.Gitea.BaseURL, h.config.VCS.Gitea.Token)

	_, err := reviewer.RunReview(ctx, h.g, prDetails, h.config, vcsClient)
	if err != nil {
		log.Printf("Code review failed for Gitea PR #%d: %v", prDetails.PRNumber, err)
		vcsClient.PostGeneralComment(ctx, prDetails.Owner, prDetails.Repo, prDetails.PRNumber, "❌ AI Review Failed: An internal error occurred.")
	}
}
