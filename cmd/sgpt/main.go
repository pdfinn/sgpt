// Package main provides the sgpt CLI command
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"sgpt/pkg/config"
	"sgpt/pkg/provider"
	"sgpt/pkg/provider/openai"
	"sgpt/pkg/transport"
)

func main() {
	// Create a cancelable context to handle interruptions
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Load configuration
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create the transport client
	client := transport.NewClient(cfg.Logger)

	// Create and register providers
	providers := make(map[string]provider.LLMProvider)

	// Register OpenAI provider
	providers["openai"] = openai.NewProvider(client, cfg)

	// TODO: Register other providers when implemented
	// providers["anthropic"] = anthropic.NewProvider(client, cfg)
	// providers["google"] = gemini.NewProvider(client, cfg)

	// Get the requested provider
	llm, ok := providers[cfg.Provider]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: provider %s not implemented\n", cfg.Provider)
		os.Exit(1)
	}

	// Create a scanner to read from stdin
	scanner := bufio.NewScanner(os.Stdin)
	buf := make([]byte, 0, 64*1024) // 64KB buffer
	scanner.Buffer(buf, 1024*1024)  // Allow up to 1MB per line

	// If we're reading from terminal and no arguments, inform the user
	stat, _ := os.Stdin.Stat()
	if (stat.Mode()&os.ModeCharDevice) != 0 && len(os.Args) <= 1 {
		fmt.Println("Reading input from stdin. Press Ctrl+D when finished.")
	}

	// Read all input
	var input string
	for scanner.Scan() {
		input += scanner.Text() + "\n"
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	// If we have command line arguments as input (non-flag), use those instead
	if args := cfg.RemainingArgs; len(args) > 0 {
		input = strings.Join(args, " ")
	}

	// If no input, nothing to do
	if input == "" && cfg.ImagePath == "" && cfg.AudioPath == "" {
		fmt.Fprintf(os.Stderr, "Error: no input provided\n")
		os.Exit(1)
	}

	// Process input in chunks based on separator
	inputs := strings.Split(input, cfg.Separator)

	for _, chunk := range inputs {
		// Skip empty chunks
		chunk = strings.TrimSpace(chunk)
		if chunk == "" && cfg.ImagePath == "" && cfg.AudioPath == "" {
			continue
		}

		// Create request for the provider
		req := provider.Request{
			Model:       cfg.Model,
			Instruction: cfg.Instruction,
			Input:       chunk,
			Temperature: cfg.Temperature,
			ImagePath:   cfg.ImagePath,
			AudioPath:   cfg.AudioPath,
			Stream:      cfg.ModelCaps.Streaming,
		}

		// Process the request
		if req.Stream {
			if err := llm.StreamComplete(ctx, req); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else {
			resp, err := llm.Complete(ctx, req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(resp.Text)
		}
	}
}
