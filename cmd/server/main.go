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

	ClientNAMe := "techCLient"
	fmt.Println("CLientNAme ", ClientNAMe)

	g, err := initGenkit(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Genkit: %v", err)
	}

	githubWebhookSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if githubWebhookSecret == "" {
		log.Fatalf("GITHUB_WEBHOOK_SECRET environment variable not set")
	}

	// The handler is now simpler and doesn't need a flow passed to it.
	githubHandler, err := webhook.NewGitHubWebhookHandler(g, cfg, githubWebhookSecret)
	if err != nil {
		log.Fatalf("Failed to create GitHub webhook handler: %v", err)
	}

	router := gin.Default()
	router.POST("/api/github/webhook", githubHandler.Handle)
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
