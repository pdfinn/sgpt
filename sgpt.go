package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

// OpenAIResponse structure to handle JSON response from OpenAI API
type OpenAIResponse struct {
	Choices []struct {
		Text    string `json:"text,omitempty"`
		Message struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"message,omitempty"`
	} `json:"choices"`
}

// Function to setup configuration using viper and pflag
func setupConfig() {
	viper.SetConfigName(".sgpt")           // Name of the configuration file without the extension
	viper.SetConfigType("yaml")            // Extension of the configuration file
	viper.AddConfigPath(".")               // First look for config in the working directory
	viper.AddConfigPath(os.Getenv("HOME")) // Fallback to the home directory

	// Setting up command line flags using Unix style single-character flags
	pflag.StringP("apiKey", "k", "", "API key for OpenAI")
	pflag.StringP("model", "m", "", "Model to use for OpenAI API")
	pflag.StringP("instruction", "i", "", "Instruction for OpenAI")
	pflag.StringP("text", "t", "", "Text to process (optional, falls back to stdin)")
	pflag.Float64P("temperature", "T", 0.5, "Temperature setting for the model")

	// Bind environment variables
	viper.BindEnv("apiKey", "SGPT_API_KEY")
	viper.BindEnv("model", "SGPT_MODEL")
	viper.BindEnv("instruction", "SGPT_INSTRUCTION")
	viper.BindEnv("temperature", "SGPT_TEMPERATURE")

	// Parsing the flags
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("Config file not found: %v", err) // Non-fatal error
		} else {
			log.Fatalf("Error reading config file: %v", err)
		}
	}
}

// Function to handle API calls to OpenAI based on model
func callOpenAI(apiKey, model, instruction, input string, temperature float64) (string, error) {
	var url string
	var jsonData []byte
	var err error

	switch model {
	case "gpt-4", "gpt-4-0314", "gpt-4-32k", "gpt-4-32k-0314", "gpt-3.5-turbo":
		url = "https://api.openai.com/v1/chat/completions"
		// Prepare JSON data for GPT-4 models
		messages := []map[string]string{
			{"role": "system", "content": instruction},
			{"role": "user", "content": input},
		}
		jsonData, err = json.Marshal(map[string]interface{}{
			"model":       model,
			"messages":    messages,
			"temperature": temperature,
			"max_tokens":  100,
			"stop":        []string{"\n"},
		})

	case "text-davinci-003", "text-davinci-002", "text-curie-001", "text-babbage-001", "text-ada-001":
		url = "https://api.openai.com/v1/completions"
		// Prepare JSON data for GPT-3 models
		prompt := instruction + " " + input
		jsonData, err = json.Marshal(map[string]interface{}{
			"model":       model,
			"prompt":      prompt,
			"temperature": temperature,
			"max_tokens":  100,
			"stop":        []string{"\n"},
		})

	case "whisper-1":
		url = "https://api.openai.com/v1/audio/transcriptions"
	default:
		return "", fmt.Errorf("unsupported model: %s", model)
	}

	if err != nil {
		return "", err
	}

	data := bytes.NewReader(jsonData)
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

	var response OpenAIResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from the API")
	}

	assistantMessage := ""
	for _, choice := range response.Choices {
		if choice.Message.Role == "assistant" {
			assistantMessage = strings.TrimSpace(choice.Message.Content)
			break
		}
		if choice.Text != "" {
			assistantMessage = strings.TrimSpace(choice.Text)
			break
		}
	}

	if assistantMessage == "" {
		return "", fmt.Errorf("no assistant message found in the API response")
	}

	return assistantMessage, nil
}

func main() {
	setupConfig() // Set up configuration

	// Fetch configurations from Viper
	apiKey := viper.GetString("apiKey")
	model := viper.GetString("model")
	instruction := viper.GetString("instruction")
	temperature := viper.GetFloat64("temperature")
	input := viper.GetString("text")

	if input == "" {
		// If no text is provided via command line, read from stdin
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			input += scanner.Text() + "\n"
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("Error reading input from stdin: %v", err)
		}
	}

	message, err := callOpenAI(apiKey, model, instruction, input, temperature)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(message) // Output only the message
}
