package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	debug bool
)

// Function to setup configuration using viper and pflag
func setupConfig() {
	viper.SetConfigName(".sgpt")           // Name of the configuration file without the extension
	viper.SetConfigType("yaml")            // Extension of the configuration file
	viper.AddConfigPath(".")               // First look for config in the working directory
	viper.AddConfigPath(os.Getenv("HOME")) // Fallback to the home directory

	// Setting up command line flags using Unix style single-character flags
	pflag.StringP("api_key", "k", "", "API key for the selected provider")
	pflag.StringP("model", "m", "", "Model to use for the API")
	pflag.StringP("instruction", "i", "", "Instruction for the model")
	pflag.Float64P("temperature", "t", 0.5, "Temperature setting for the model")
	pflag.StringP("separator", "s", "\n", "Separator character for input")
	pflag.BoolP("debug", "d", false, "Enable debug output")
	pflag.StringP("provider", "p", "openai", "Provider to use for API (openai, anthropic, google)")

	// New flags for image and audio inputs
	pflag.StringP("image", "g", "", "Path or URL to an image file")
	pflag.StringP("audio", "a", "", "Path to an audio file")

	// Bind environment variables
	viper.BindEnv("api_key", "SGPT_API_KEY")
	viper.BindEnv("model", "SGPT_MODEL")
	viper.BindEnv("instruction", "SGPT_INSTRUCTION")
	viper.BindEnv("temperature", "SGPT_TEMPERATURE")
	viper.BindEnv("separator", "SGPT_SEPARATOR")
	viper.BindEnv("debug", "SGPT_DEBUG")
	viper.BindEnv("provider", "SGPT_PROVIDER")
	viper.BindEnv("image", "SGPT_IMAGE")
	viper.BindEnv("audio", "SGPT_AUDIO")

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

	debug = viper.GetBool("debug")
}

// Function to handle API calls based on provider
func callAPI(provider, apiKey, model, instruction, input, imagePath, audioPath string, temperature float64) (string, error) {
	switch provider {
	case "openai":
		return callOpenAI(apiKey, model, instruction, input, imagePath, audioPath, temperature)
	case "anthropic":
		return callAnthropic(apiKey, model, instruction, input, imagePath, audioPath, temperature)
	case "google":
		return callGoogle(apiKey, model, instruction, input, imagePath, audioPath, temperature)
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

// Function to handle API calls to OpenAI based on model
func callOpenAI(apiKey, model, instruction, input, imagePath, audioPath string, temperature float64) (string, error) {
	var url string
	var jsonData []byte
	var err error

	switch model {
	case "gpt-4o":
		url = "https://api.openai.com/v1/chat/completions"

		var messages []map[string]interface{}

		// Add system message if instruction is provided
		if instruction != "" {
			messages = append(messages, map[string]interface{}{
				"role":    "system",
				"content": instruction,
			})
		}

		// Build user content
		var userContent []map[string]interface{}

		// Add text input
		if input != "" {
			userContent = append(userContent, map[string]interface{}{
				"type": "text",
				"text": input,
			})
		}

		// Add image content
		if imagePath != "" {
			var imageURL string
			if strings.HasPrefix(imagePath, "http://") || strings.HasPrefix(imagePath, "https://") {
				imageURL = imagePath
			} else {
				// Read image file and encode as base64
				imageBytes, err := ioutil.ReadFile(imagePath)
				if err != nil {
					return "", fmt.Errorf("failed to read image file: %v", err)
				}
				base64Image := base64.StdEncoding.EncodeToString(imageBytes)
				imageURL = "data:image/png;base64," + base64Image
			}

			userContent = append(userContent, map[string]interface{}{
				"type": "image_url",
				"image_url": map[string]string{
					"url": imageURL,
				},
			})
		}

		// Add audio content
		if audioPath != "" {
			// Read audio file and encode as base64
			audioBytes, err := ioutil.ReadFile(audioPath)
			if err != nil {
				return "", fmt.Errorf("failed to read audio file: %v", err)
			}
			base64Audio := base64.StdEncoding.EncodeToString(audioBytes)
			audioData := "data:audio/wav;base64," + base64Audio

			userContent = append(userContent, map[string]interface{}{
				"type": "audio",
				"audio": map[string]string{
					"data": audioData,
				},
			})
		}

		// Add the user message
		messages = append(messages, map[string]interface{}{
			"role":    "user",
			"content": userContent,
		})

		// Prepare JSON data
		jsonData, err = json.Marshal(map[string]interface{}{
			"model":       model,
			"messages":    messages,
			"temperature": temperature,
			"max_tokens":  100,
		})
		if err != nil {
			return "", err
		}

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

	default:
		return "", fmt.Errorf("unsupported model: %s", model)
	}

	if err != nil {
		return "", err
	}

	if debug {
		log.Printf("Sending request to %s with data: %s", url, string(jsonData))
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

	if debug {
		log.Printf("Received response: %s", string(body))
	}

	// OpenAIResponse structure to handle JSON response from OpenAI API
	var response struct {
		Choices []struct {
			Text    string `json:"text,omitempty"`
			Message struct {
				Role    string `json:"role,omitempty"`
				Content string `json:"content,omitempty"`
			} `json:"message,omitempty"`
		} `json:"choices"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from the API")
	}

	assistantMessage := ""
	for _, choice := range response.Choices {
		if choice.Message.Content != "" {
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

// Function to handle API calls to Anthropic
func callAnthropic(apiKey, model, instruction, input, imagePath, audioPath string, temperature float64) (string, error) {
	// Placeholder implementation for Anthropic API
	return "", fmt.Errorf("Anthropic API integration is not implemented")
}

// Function to handle API calls to Google Gemini
func callGoogle(apiKey, model, instruction, input, imagePath, audioPath string, temperature float64) (string, error) {
	// Placeholder implementation for Google Gemini API
	return "", fmt.Errorf("Google Gemini API integration is not implemented")
}

func main() {
	setupConfig() // Set up configuration

	// Fetch configurations from Viper
	apiKey := viper.GetString("api_key")
	model := viper.GetString("model")
	instruction := viper.GetString("instruction")
	temperature := viper.GetFloat64("temperature")
	separator := viper.GetString("separator")
	provider := viper.GetString("provider")

	// New configurations for image and audio
	imagePath := viper.GetString("image")
	audioPath := viper.GetString("audio")

	if apiKey == "" {
		log.Fatal("API key is required. Please provide it via --api_key flag, SGPT_API_KEY environment variable, or config file.")
	}

	if model == "" {
		// Set default model based on provider
		switch provider {
		case "openai":
			model = "gpt-3.5-turbo"
		case "anthropic":
			model = "claude-v1"
		case "google":
			model = "gemini-medium"
		}
	}

	var inputData string
	if pflag.NArg() > 0 {
		// Process additional arguments as input
		inputData = strings.Join(pflag.Args(), " ")
	} else {
		// Read from stdin if no arguments are provided
		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("Error reading input from stdin: %v", err)
		}
		inputData = string(data)
	}

	inputs := strings.Split(inputData, separator)

	for _, input := range inputs {
		input = strings.TrimSpace(input)
		if input == "" && imagePath == "" && audioPath == "" {
			continue
		}
		message, err := callAPI(provider, apiKey, model, instruction, input, imagePath, audioPath, temperature)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(message) // Output only the message
	}
}
