package main

import (
	"os"
	"testing"
)

func TestCallOpenAI(t *testing.T) {
	apiKey := os.Getenv("SGPT_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping TestCallOpenAI. Set the SGPT_API_KEY environment variable to run this test.")
	}

	instruction := "Answer the question."
	input := "What is the capital of France?"
	temperature := 0.5
	model := "gpt-4"

	response, err := callOpenAI(apiKey, instruction, input, temperature, model)
	if err != nil {
		t.Errorf("Unexpected error calling OpenAI API: %v", err)
	}

	if response == "" {
		t.Error("Empty response from callOpenAI")
	}
}
