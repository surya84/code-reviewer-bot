package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the application's configuration.
type Config struct {
	VCS              VCSConfig `yaml:"vcs"`
	LLM              LLMConfig `yaml:"llm"`
	ReviewPromptFile string    `yaml:"review_prompt_file"`
	// This now holds the fully assembled prompt after loading.
	ReviewPrompt string `yaml:"review_prompt"`
}

// VCSConfig holds configuration for the version control system.
type VCSConfig struct {
	Provider string       `yaml:"provider"`
	GitHub   GitHubConfig `yaml:"github"`
	Gitea    GiteaConfig  `yaml:"gitea"`
}

// GitHubConfig holds GitHub-specific settings.
type GitHubConfig struct {
	Token string `yaml:"token"`
}

// GiteaConfig holds Gitea-specific settings.
type GiteaConfig struct {
	BaseURL string `yaml:"base_url"`
	Token   string `yaml:"token"`
}

// LLMConfig holds configuration for the language model.
type LLMConfig struct {
	Provider  string `yaml:"provider"`
	ModelName string `yaml:"model_name"`
	GoogleAI  struct {
		APIKey string `yaml:"api_key"`
	} `yaml:"googleai"`
	OpenAI struct {
		APIKey string `yaml:"api_key"`
	} `yaml:"openai"`
}

// LoadConfig reads the configuration, loads the base prompt from a file,
// and assembles the final review prompt.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Expand environment variables like ${VAR_NAME}
	expandedData := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expandedData), &cfg); err != nil {
		return nil, err
	}

	// Check if a prompt file is specified.
	if cfg.ReviewPromptFile == "" {
		return nil, fmt.Errorf("'review_prompt_file' must be specified in config.yaml")
	}

	// Read the base prompt from the specified file.
	basePromptBytes, err := os.ReadFile(cfg.ReviewPromptFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read review prompt file '%s': %w", cfg.ReviewPromptFile, err)
	}
	basePrompt := string(basePromptBytes)

	// Combine the base prompt with the suffix from the yaml file.
	var finalPrompt strings.Builder
	finalPrompt.WriteString(basePrompt)
	finalPrompt.WriteString("\n\n") // Add a separator
	finalPrompt.WriteString(cfg.ReviewPrompt)

	// Overwrite the ReviewPrompt field with the fully assembled prompt.
	cfg.ReviewPrompt = finalPrompt.String()

	return &cfg, nil
}
