// Package provider defines the interface for LLM providers
package provider

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"sgpt/pkg/config"
)

// Common errors
var (
	ErrUnsupportedModel     = errors.New("unsupported model")
	ErrAPIRequestFailed     = errors.New("API request failed")
	ErrNoResponseGenerated  = errors.New("no response generated")
	ErrInvalidConfiguration = errors.New("invalid configuration")
)

// Request represents a completion request to an LLM provider
type Request struct {
	// Core request parameters
	Instruction string
	Input       string
	Temperature float64
	Model       string

	// Optional multimodal inputs
	ImagePath string
	AudioPath string

	// Response streaming
	Stream bool
}

// Response represents a completion response from an LLM provider
type Response struct {
	// The generated text
	Text string

	// Raw response (could be useful for debugging)
	Raw []byte
}

// LLMProvider defines the interface that all provider implementations must satisfy
type LLMProvider interface {
	// Complete sends a request to the provider's API and returns the response
	Complete(ctx context.Context, req Request) (Response, error)

	// StreamComplete sends a request and streams the response tokens as they arrive
	StreamComplete(ctx context.Context, req Request) error
}

// ProviderRegistry tracks available providers
type ProviderRegistry struct {
	providers map[string]LLMProvider
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]LLMProvider),
	}
}

// Register adds a provider to the registry
func (r *ProviderRegistry) Register(name string, provider LLMProvider) {
	r.providers[name] = provider
}

// Get retrieves a provider by name
func (r *ProviderRegistry) Get(name string) (LLMProvider, error) {
	provider, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", name)
	}
	return provider, nil
}

// Helper functions for multimodal content

// LoadImageAsBase64 loads an image file and returns it as a base64 encoded string
func LoadImageAsBase64(path string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		// Return URL directly if it's a remote image
		return path, nil
	}

	// Read image file
	imageBytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read image file: %w", err)
	}

	// Encode as base64
	base64Image := base64.StdEncoding.EncodeToString(imageBytes)
	return "data:image/png;base64," + base64Image, nil
}

// LoadAudioAsBase64 loads an audio file and returns it as a base64 encoded string
func LoadAudioAsBase64(path string) (string, error) {
	// Read audio file
	audioBytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read audio file: %w", err)
	}

	// Encode as base64
	base64Audio := base64.StdEncoding.EncodeToString(audioBytes)
	return "data:audio/wav;base64," + base64Audio, nil
}

// New creates a provider based on the configuration
func New(cfg config.Config) (LLMProvider, error) {
	// This will be implemented when concrete providers are added
	// Returning an error for now to indicate it's not fully implemented
	return nil, fmt.Errorf("provider %s not yet implemented", cfg.Provider)
}
