package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the application's configuration.
type Config struct {
	VCS          VCSConfig `yaml:"vcs"`
	LLM          LLMConfig `yaml:"llm"`
	ReviewPrompt string    `yaml:"review_prompt"`
}

// VCSConfig holds configuration for the version control system.
type VCSConfig struct {
	Provider string       `yaml:"provider"`
	GitHub   GitHubConfig `yaml:"github"`
	Gitty    GittyConfig  `yaml:"gitty"`
}

// GitHubConfig holds GitHub-specific settings.
type GitHubConfig struct {
	Token string `yaml:"token"`
}

// GittyConfig holds Gitty-specific settings (for demonstration).
type GittyConfig struct {
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

// LoadConfig reads the configuration from the given path and expands environment variables.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	expandedData := os.ExpandEnv(string(data))
	var cfg Config
	err = yaml.Unmarshal([]byte(expandedData), &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
