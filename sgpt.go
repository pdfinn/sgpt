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

//TODO add support for config file
//TODO add support for Whisper
//TODO and general file system operations

type OpenAIResponse struct {
	Choices []struct {
		Text    string `json:"text,omitempty"`
		Message struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"message,omitempty"`
	} `json:"choices"`
}

var debug *bool

func init() {
	envDebug := os.Getenv("SGPT_DEBUG")
	debug = flag.Bool("d", parseBoolWithDefault(envDebug, false), "Enable debug output")
}

func main() {
	// Default values
	defaultTemperature := 0.5
	defaultModel := "gpt-4"

	// Check environment variables
	envApiKey := os.Getenv("SGPT_API_KEY")
	envInstruction := os.Getenv("SGPT_INSTRUCTION")
	envTemperature, err := strconv.ParseFloat(os.Getenv("SGPT_TEMPERATURE"), 64)
	if err != nil {
		envTemperature = defaultTemperature
	}
	envModel := os.Getenv("SGPT_MODEL")
	envSeparator := os.Getenv("SGPT_SEPARATOR")

	// Command line arguments
	apiKey := flag.String("k", envApiKey, "OpenAI API key")
	instruction := flag.String("i", envInstruction, "Instruction for the GPT model")
	temperature := flag.Float64("t", envTemperature, "Temperature for the GPT model")
	model := flag.String("m", envModel, "GPT model to use")
	separator := flag.String("s", envSeparator, "Separator character for input")
	flag.Parse()

	// Use default values if neither flags nor environment variables are set
	if *model == "" {
		*model = defaultModel
	}

	if *apiKey == "" {
		log.Fatal("API key is required")
	}

	// Read input from stdin continuously
	reader := bufio.NewReader(os.Stdin)
	var inputBuffer strings.Builder

	for {
		inputChar, _, err := reader.ReadRune()
		if err != nil {
			break
		}

		if string(inputChar) == *separator {
			input := inputBuffer.String()
			inputBuffer.Reset()

			response, err := callOpenAI(*apiKey, *instruction, input, *temperature, *model)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println(response)
		} else {
			inputBuffer.WriteRune(inputChar)
		}
	}
}

func debugOutput(debug bool, format string, a ...interface{}) {
	if debug {
		log.Printf(format, a...)
	}
}

func parseFloatWithDefault(value string, defaultValue float64) float64 {
	if value == "" {
		return defaultValue
	}
	parsedValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		log.Printf("Warning: Failed to parse float value: %v", err)
		return defaultValue
	}
	return parsedValue
}

func parseBoolWithDefault(value string, defaultValue bool) bool {
	if value == "" {
		return defaultValue
	}
	parsedValue, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("Warning: Failed to parse bool value: %v", err)
		return defaultValue
	}
	return parsedValue
}

func callOpenAI(apiKey, instruction, input string, temperature float64, model string) (string, error) {
	var url string
	var jsonData []byte
	var err error

	switch model {
	case "gpt-4", "gpt-4-0314", "gpt-4-32k", "gpt-4-32k-0314", "gpt-3.5-turbo", "gpt-3.5-turbo-0301":
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

	debugOutput(*debug, "API response: %s\n", string(body))

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
