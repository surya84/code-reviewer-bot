package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/gin-gonic/gin"
	"github.com/surya84/code-reviewer-bot/config"
	"github.com/surya84/code-reviewer-bot/constants"
	"github.com/surya84/code-reviewer-bot/internal/webhook"
)

func main() {
	log.Println("Starting AI Code Reviewer in server mode with Gin...")
	ctx := context.Background()

	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	g, err := initGenkit(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Genkit: %v", err)
	}

	router := gin.Default()

	// set up the GitHub handler.
	githubWebhookSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubWebhookSecret != "" && githubToken != "" {
		log.Println("GitHub credentials found. Initializing GitHub handler...")
		githubHandler, err := webhook.NewGitHubWebhookHandler(g, cfg, githubWebhookSecret)
		if err != nil {
			log.Printf("WARNING: Could not create GitHub webhook handler: %v", err)
		} else {
			router.POST("/api/github/webhook", githubHandler.Handle)
			log.Println("✅ GitHub webhook endpoint (/api/github/webhook) is active.")
		}
	} else {
		log.Println("INFO: GITHUB_WEBHOOK_SECRET or GITHUB_TOKEN not found. Skipping GitHub handler setup.")
	}

	// set up the Gitea handler.
	giteaWebhookSecret := os.Getenv("GITEA_WEBHOOK_SECRET")
	giteaToken := os.Getenv("GITEA_TOKEN")
	if giteaWebhookSecret != "" && giteaToken != "" {
		log.Println("Gitea credentials found. Initializing Gitea handler...")
		giteaHandler, err := webhook.NewGiteaWebhookHandler(g, cfg, giteaWebhookSecret)
		if err != nil {
			log.Printf("WARNING: Could not create Gitea webhook handler: %v", err)
		} else {
			router.POST("/api/gitea/webhook", giteaHandler.Handle)
			log.Println("✅ Gitea webhook endpoint (/api/gitea/webhook) is active.")
		}
	} else {
		log.Println("INFO: GITEA_WEBHOOK_SECRET or GITEA_TOKEN not found. Skipping Gitea handler setup.")
	}

	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "AI Code Reviewer Bot is running.")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Listening for webhooks on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start Gin server: %v", err)
	}
}

// initGenkit initializes the Genkit instance and loads the appropriate LLM plugin.
func initGenkit(ctx context.Context, cfg *config.Config) (*genkit.Genkit, error) {
	var plugin genkit.Plugin
	switch cfg.LLM.Provider {
	case constants.GOOGLEAI:
		plugin = &googlegenai.GoogleAI{APIKey: cfg.LLM.GoogleAI.APIKey}
	case constants.OPENAI:
		plugin = &openai.OpenAI{APIKey: cfg.LLM.OpenAI.APIKey}
	default:
		return nil, fmt.Errorf("unsupported LLM provider in config: %s", cfg.LLM.Provider)
	}
	return genkit.Init(ctx, genkit.WithPlugins(plugin))
}
