package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type OpenAIResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

var debug *bool

func init() {
	debug = flag.Bool("d", false, "Enable debug output")
}

func main() {
	// Command line arguments
	apiKey := flag.String("k", "", "OpenAI API key")
	instruction := flag.String("i", "", "Instruction for the GPT model")
	temperature := flag.Float64("t", 0.5, "Temperature for the GPT model")
	model := flag.String("m", "gpt-4", "GPT model to use")
	flag.Parse()

	if *apiKey == "" {
		log.Fatal("API key is required")
	}

	// Read input from stdin
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	// Call OpenAI API
	response, err := callOpenAI(*apiKey, *instruction, input, *temperature, *model)
	if err != nil {
		log.Fatal(err)
	}

	// Print the result
	fmt.Println(response)
}

func debugOutput(debug bool, format string, a ...interface{}) {
	if debug {
		log.Printf(format, a...)
	}
}

func callOpenAI(apiKey, instruction, input string, temperature float64, model string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"

	// Prepare JSON data
	messages := []map[string]string{
		{"role": "system", "content": instruction},
		{"role": "user", "content": input},
	}

	jsonData, err := json.Marshal(map[string]interface{}{
		"model":       model,
		"messages":    messages,
		"temperature": temperature,
		"max_tokens":  100,
		"stop":        []string{"\n"},
	})

	if err != nil {
		return "", err
	}

	data := strings.NewReader(string(jsonData))

	req, err := http.NewRequest("POST", url, data)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	debugOutput(*debug, "API response: %s\n", string(body)) // Add this line to print the raw API response

	var openAIResponse OpenAIResponse
	err = json.Unmarshal(body, &openAIResponse)
	if err != nil {
		return "", err
	}

	if len(openAIResponse.Choices) == 0 {
		debugOutput(*debug, "API response: %s\n", string(body))
		debugOutput(*debug, "HTTP status code: %s\n", strconv.Itoa(resp.StatusCode))
		return "", fmt.Errorf("no choices returned from the API")
	}

	assistantMessage := ""
	for _, choice := range openAIResponse.Choices {
		if choice.Message.Role == "assistant" {
			assistantMessage = strings.TrimSpace(choice.Message.Content)
			break
		}
	}

	if assistantMessage == "" {
		return "", fmt.Errorf("no assistant message found in the API response")
	}

	return assistantMessage, nil
}
