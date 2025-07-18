package reviewer

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/surya84/code-reviewer-bot/config"
	"github.com/surya84/code-reviewer-bot/constants"
	"github.com/surya84/code-reviewer-bot/internal/reviewer"
	"github.com/surya84/code-reviewer-bot/pkg/vcs"
)

var (
	configPath string
	repoOwner  string
	repoName   string
	prNumber   int
)

var rootCmd = &cobra.Command{
	Use:   "code-reviewer-bot",
	Short: "AI-powered Code Reviewer",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			log.Fatalf("❌ Failed to load config: %v", err)
		}

		g, err := initGenkit(ctx, cfg)
		if err != nil {
			log.Fatalf("❌ Failed to initialize Genkit: %v", err)
		}

		prDetails, err := getPRDetails()
		if err != nil {
			log.Fatalf("❌ Failed to get PR details: %v", err)
		}

		vcsClient, err := vcs.NewVCSClient(ctx, &cfg.VCS)
		if err != nil {
			log.Fatalf("❌ Failed to create VCS client: %v", err)
		}

		result, err := reviewer.RunReview(ctx, g, prDetails, cfg, vcsClient)
		if err != nil {
			log.Fatalf("❌ Code review process failed: %v", err)
		}

		log.Printf("✅ Process finished: %s", result)
	},
}

func init() {
	rootCmd.Flags().StringVar(&configPath, "config", "/app/config/config.yaml", "Path to config.yaml")
	rootCmd.Flags().StringVar(&repoOwner, "repo-owner", "", "Repository owner (overrides env)")
	rootCmd.Flags().StringVar(&repoName, "repo-name", "", "Repository name (overrides env)")
	rootCmd.Flags().IntVar(&prNumber, "pr-number", 0, "PR number (overrides env)")
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func initGenkit(ctx context.Context, cfg *config.Config) (*genkit.Genkit, error) {
	var plugin genkit.Plugin
	switch cfg.LLM.Provider {
	case constants.GOOGLEAI:
		plugin = &googlegenai.GoogleAI{APIKey: cfg.LLM.GoogleAI.APIKey}
	case constants.OPENAI:
		plugin = &openai.OpenAI{APIKey: cfg.LLM.OpenAI.APIKey}
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.LLM.Provider)
	}
	return genkit.Init(ctx, genkit.WithPlugins(plugin))
}

func getPRDetails() (*reviewer.PRDetails, error) {
	repoSlug := os.Getenv("GITHUB_REPOSITORY")
	if repoSlug == "" {
		return nil, fmt.Errorf("GITHUB_REPOSITORY is not set")
	}
	parts := strings.Split(repoSlug, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid GITHUB_REPOSITORY format: got '%s', expected 'owner/repo'", repoSlug)
	}

	prStr := os.Getenv("PR_NUMBER")
	if prStr == "" {
		return nil, fmt.Errorf("PR_NUMBER is not set")
	}
	prNumber, err := strconv.Atoi(prStr)
	if err != nil {
		return nil, fmt.Errorf("invalid PR_NUMBER: expected integer, got '%s'", prStr)
	}

	return &reviewer.PRDetails{
		Owner:    parts[0],
		Repo:     parts[1],
		PRNumber: prNumber,
	}, nil
}

func split(s, sep string) []string {
	var result []string
	for _, part := range os.Environ() {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
