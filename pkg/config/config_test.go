package config

import (
	"os"
	"testing"
)

func TestConfigPrecedence(t *testing.T) {
	// Create a temporary config file
	tmpfile, err := os.CreateTemp("", "sgpt-test-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	// Write test config
	yamlContent := []byte(`
api_key: yaml-key
model: yaml-model
instruction: yaml-instruction
temperature: 0.7
separator: "|"
debug: false
provider: openai
`)
	if _, err := tmpfile.Write(yamlContent); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Set environment variables
	os.Setenv("SGPT_API_KEY", "env-key")
	os.Setenv("SGPT_MODEL", "gpt-3.5-turbo")
	os.Setenv("SGPT_TEMPERATURE", "0.8")
	defer func() {
		os.Unsetenv("SGPT_API_KEY")
		os.Unsetenv("SGPT_MODEL")
		os.Unsetenv("SGPT_TEMPERATURE")
	}()

	// Define test cases
	tests := []struct {
		name     string
		args     []string
		expected Config
	}{
		{
			name: "CLI flags take precedence",
			args: []string{
				"--api_key", "flag-key",
				"--model", "gpt-4",
				"--temperature", "0.9",
			},
			expected: Config{
				APIKey:      "flag-key",
				Model:       "gpt-4",
				Provider:    "openai", // Default from code
				Instruction: "",       // Empty because no env or flag
				Temperature: 0.9,      // From flag
				Separator:   "\n",     // Default
				Debug:       false,    // Default
			},
		},
		{
			name: "Environment takes precedence over config file",
			args: []string{},
			expected: Config{
				APIKey:      "env-key",
				Model:       "gpt-3.5-turbo",
				Provider:    "openai", // Default from code
				Instruction: "",       // Empty because no env or flag
				Temperature: 0.8,      // From env
				Separator:   "\n",     // Default
				Debug:       false,    // Default
			},
		},
	}

	// For simplicity in testing, we'll mock the viper functionality
	// by just testing the configuration precedence logic directly
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Handle args to simulate command line parameters
			cfg, err := Load(tt.args)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			// Check that values match expected
			if cfg.APIKey != tt.expected.APIKey {
				t.Errorf("APIKey = %v, want %v", cfg.APIKey, tt.expected.APIKey)
			}
			if cfg.Model != tt.expected.Model {
				t.Errorf("Model = %v, want %v", cfg.Model, tt.expected.Model)
			}
			if cfg.Temperature != tt.expected.Temperature {
				t.Errorf("Temperature = %v, want %v", cfg.Temperature, tt.expected.Temperature)
			}
			if cfg.Provider != tt.expected.Provider {
				t.Errorf("Provider = %v, want %v", cfg.Provider, tt.expected.Provider)
			}
		})
	}
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "Valid config",
			config: Config{
				APIKey:      "test-key",
				Provider:    "openai",
				Model:       "gpt-4",
				Temperature: 0.5,
			},
			expectError: false,
		},
		{
			name: "Missing API key",
			config: Config{
				Provider:    "openai",
				Model:       "gpt-4",
				Temperature: 0.5,
			},
			expectError: true,
		},
		{
			name: "Invalid provider",
			config: Config{
				APIKey:      "test-key",
				Provider:    "invalid",
				Model:       "gpt-4",
				Temperature: 0.5,
			},
			expectError: true,
		},
		{
			name: "Invalid model",
			config: Config{
				APIKey:      "test-key",
				Provider:    "openai",
				Model:       "invalid-model",
				Temperature: 0.5,
			},
			expectError: true,
		},
		{
			name: "Temperature too high",
			config: Config{
				APIKey:      "test-key",
				Provider:    "openai",
				Model:       "gpt-4",
				Temperature: 1.5,
			},
			expectError: true,
		},
		{
			name: "Image with non-multimodal model",
			config: Config{
				APIKey:      "test-key",
				Provider:    "openai",
				Model:       "text-davinci-003", // Not multimodal
				Temperature: 0.5,
				ImagePath:   "image.png",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.expectError {
				t.Errorf("Validate() error = %v, expectError = %v", err, tt.expectError)
			}
		})
	}
}
