package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

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
	debug = pflag.Bool("d", parseBoolWithDefault(envDebug, false), "Enable debug output")
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
	envDebug := parseBoolWithDefault(os.Getenv("SGPT_DEBUG"), false)

	// Command line arguments
	apiKey := pflag.StringP("key", "k", envApiKey, "OpenAI API key")
	instruction := pflag.StringP("instruction", "i", envInstruction, "Instruction for the GPT model")
	temperature := pflag.Float64P("temperature", "t", envTemperature, "Temperature for the GPT model")
	model := pflag.StringP("model", "m", envModel, "GPT model to use")
	defaulSeparator := "\n"
	separator := pflag.StringP("separator", "s", envSeparator, "Separator character for input")
	if *separator == "" {
		*separator = defaulSeparator
	}
	debug = pflag.BoolP("debug", "d", envDebug, "Enable debug output")
	pflag.Parse()

	// Read the configuration file
	viper.SetConfigName("sgpt")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.sgpt")
	viper.SetConfigType("yaml")

	err = viper.ReadInConfig()
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		log.Printf("Warning: Config file not found: %v", err)
	} else if err != nil {
		log.Printf("Warning: Error reading config file: %v", err)
	}

	// Set default values and bind configuration values to flags
	viper.SetDefault("model", defaultModel)
	viper.SetDefault("temperature", defaultTemperature)
	viper.BindPFlag("api_key", pflag.Lookup("k"))
	viper.BindPFlag("instruction", pflag.Lookup("i"))
	viper.BindPFlag("model", pflag.Lookup("m"))
	viper.BindPFlag("temperature", pflag.Lookup("t"))
	viper.BindPFlag("separator", pflag.Lookup("s"))
	viper.BindPFlag("debug", pflag.Lookup("d"))

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
		if err == io.EOF {
			input := inputBuffer.String()
			if input != "" {
				response, err := callOpenAI(*apiKey, *instruction, input, *temperature, *model)
				if err != nil {
					log.Fatal(err)
				}
				fmt.Println(response)
			}
			break
		}
		if err != nil {
			log.Fatal(err)
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
