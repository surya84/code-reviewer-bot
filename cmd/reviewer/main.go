package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/surya84/code-reviewer-bot/config"
	"github.com/surya84/code-reviewer-bot/internal/reviewer"
	"github.com/surya84/code-reviewer-bot/pkg/vcs"
)

func main() {
	ctx := context.Background()
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ClientNAme := "user name"
	fmt.Println("Client Name: ", ClientNAme)

	g, err := initGenkit(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Genkit: %v", err)
	}

	prDetails, err := getPRDetailsFromEnv()
	if err != nil {
		log.Fatalf("Failed to get PR details: %v", err)
	}

	vcsClient, err := vcs.NewVCSClient(ctx, &cfg.VCS)
	if err != nil {
		log.Fatalf("Failed to create VCS client: %v", err)
	}

	// Directly call the review orchestration function.
	result, err := reviewer.RunReview(ctx, g, prDetails, cfg, vcsClient)
	if err != nil {
		log.Fatalf("Code review process failed: %v", err)
	}

	log.Printf("Process finished successfully: %s", result)
}

func initGenkit(ctx context.Context, cfg *config.Config) (*genkit.Genkit, error) {
	var plugin genkit.Plugin
	switch cfg.LLM.Provider {
	case "googleai":
		plugin = &googlegenai.GoogleAI{APIKey: cfg.LLM.GoogleAI.APIKey}
	case "openai":
		plugin = &openai.OpenAI{APIKey: cfg.LLM.OpenAI.APIKey}
	default:
		return nil, fmt.Errorf("unsupported LLM provider in config: %s", cfg.LLM.Provider)
	}
	return genkit.Init(ctx, genkit.WithPlugins(plugin))
}

func getPRDetailsFromEnv() (*reviewer.PRDetails, error) {
	repoSlug := os.Getenv("GITHUB_REPOSITORY")
	if repoSlug == "" {
		repoOwner := os.Getenv("REPO_OWNER")
		repoName := os.Getenv("REPO_NAME")
		if repoOwner == "" || repoName == "" {
			return nil, fmt.Errorf("GITHUB_REPOSITORY (or REPO_OWNER/REPO_NAME) env var not set")
		}
		repoSlug = fmt.Sprintf("%s/%s", repoOwner, repoName)
	}
	parts := strings.Split(repoSlug, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid GITHUB_REPOSITORY format: %s", repoSlug)
	}
	prNumberStr := os.Getenv("PR_NUMBER")
	if prNumberStr == "" {
		return nil, fmt.Errorf("PR_NUMBER env var not set")
	}
	prNumber, err := strconv.Atoi(prNumberStr)
	if err != nil {
		return nil, fmt.Errorf("invalid PR_NUMBER: %w", err)
	}
	return &reviewer.PRDetails{
		Owner:    parts[0],
		Repo:     parts[1],
		PRNumber: prNumber,
	}, nil
}
