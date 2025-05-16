// Package config provides configuration handling for sgpt
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// ModelCapabilities describes what a model can do
type ModelCapabilities struct {
	Multimodal bool
	Streaming  bool
}

// Config holds all configuration for the application
type Config struct {
	// API credentials and settings
	APIKey    string
	Provider  string
	Model     string
	ModelCaps ModelCapabilities

	// Request behavior
	Instruction string
	Temperature float64
	Separator   string

	// Input data
	ImagePath string
	AudioPath string

	// Operational settings
	Debug bool

	// Logger instance
	Logger *slog.Logger

	// Non-flag arguments remaining after parsing
	RemainingArgs []string
}

var (
	// Predefined model capabilities
	modelCapabilities = map[string]ModelCapabilities{
		// OpenAI models
		"gpt-4o":           {Multimodal: true, Streaming: true},
		"gpt-4":            {Multimodal: false, Streaming: true},
		"gpt-4-0314":       {Multimodal: false, Streaming: true},
		"gpt-4-32k":        {Multimodal: false, Streaming: true},
		"gpt-4-32k-0314":   {Multimodal: false, Streaming: true},
		"gpt-3.5-turbo":    {Multimodal: false, Streaming: true},
		"text-davinci-003": {Multimodal: false, Streaming: false},
		"text-davinci-002": {Multimodal: false, Streaming: false},
		"text-curie-001":   {Multimodal: false, Streaming: false},
		"text-babbage-001": {Multimodal: false, Streaming: false},
		"text-ada-001":     {Multimodal: false, Streaming: false},

		// Anthropic models
		"claude-v1":   {Multimodal: false, Streaming: true},
		"claude-v1.2": {Multimodal: false, Streaming: true},

		// Google models
		"gemini-medium": {Multimodal: true, Streaming: true},
		"gemini-large":  {Multimodal: true, Streaming: true},
	}
)

// Load reads the configuration from flags, environment, and config file
func Load(args []string) (Config, error) {
	// Initialize viper
	v := viper.New()
	v.SetConfigName("sgpt")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath(filepath.Join(os.Getenv("HOME")))

	// Create flag set
	flags := pflag.NewFlagSet("sgpt", pflag.ContinueOnError)

	// Define flags
	flags.StringP("api_key", "k", "", "API key for the selected provider")
	flags.StringP("model", "m", "", "Model to use for the API")
	flags.StringP("instruction", "i", "", "Instruction for the model")
	flags.Float64P("temperature", "t", 0.5, "Temperature setting for the model")
	flags.StringP("separator", "s", "\n", "Separator character for input")
	flags.BoolP("debug", "d", false, "Enable debug output")
	flags.StringP("provider", "p", "openai", "Provider to use for API (openai, anthropic, google)")
	flags.StringP("image", "g", "", "Path or URL to an image file")
	flags.StringP("audio", "a", "", "Path to an audio file")

	// Parse flags
	if err := flags.Parse(args); err != nil {
		return Config{}, err
	}

	// Bind flags to viper
	if err := v.BindPFlags(flags); err != nil {
		return Config{}, err
	}

	// Bind environment variables
	v.SetEnvPrefix("SGPT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file (non-fatal if not found)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return Config{}, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Get values from viper
	cfg := Config{
		APIKey:        v.GetString("api_key"),
		Provider:      v.GetString("provider"),
		Model:         v.GetString("model"),
		Instruction:   v.GetString("instruction"),
		Temperature:   v.GetFloat64("temperature"),
		Separator:     v.GetString("separator"),
		Debug:         v.GetBool("debug"),
		ImagePath:     v.GetString("image"),
		AudioPath:     v.GetString("audio"),
		RemainingArgs: flags.Args(), // Store remaining arguments after flags
	}

	// Configure logger
	logLevel := slog.LevelInfo
	if cfg.Debug {
		logLevel = slog.LevelDebug
	}

	// Create structured logger
	cfg.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// API key is required
	if c.APIKey == "" {
		return errors.New("API key is required. Please provide it via --api_key flag, SGPT_API_KEY environment variable, or config file")
	}

	// Check provider
	switch c.Provider {
	case "openai", "anthropic", "google":
		// Valid provider
	default:
		return fmt.Errorf("unsupported provider: %s", c.Provider)
	}

	// Set default model if not specified
	if c.Model == "" {
		switch c.Provider {
		case "openai":
			c.Model = "gpt-3.5-turbo"
		case "anthropic":
			c.Model = "claude-v1"
		case "google":
			c.Model = "gemini-medium"
		}
	}

	// Look up model capabilities
	caps, ok := modelCapabilities[c.Model]
	if !ok {
		return fmt.Errorf("unsupported model: %s", c.Model)
	}
	c.ModelCaps = caps

	// Check multimodal constraints
	if (c.ImagePath != "" || c.AudioPath != "") && !caps.Multimodal {
		return fmt.Errorf("model %s does not support multimodal inputs (image/audio)", c.Model)
	}

	// Validate temperature range
	if c.Temperature < 0 || c.Temperature > 1.0 {
		return fmt.Errorf("temperature must be between 0 and 1, got %f", c.Temperature)
	}

	return nil
}
