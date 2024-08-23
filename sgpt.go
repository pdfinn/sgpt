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
	// Set default values for fallback
	defaultModel := "gpt-3.5-turbo"
	defaultTemperature := 0.5

	// Initialize flags with environment variables as defaults
	apiKey := pflag.StringP("api_key", "k", os.Getenv("SGPT_API_KEY"), "OpenAI API key")
	instruction := pflag.StringP("instruction", "i", os.Getenv("SGPT_INSTRUCTION"), "Instruction for the GPT model")
	temperature := pflag.Float64P("temperature", "t", parseFloatWithDefault(os.Getenv("SGPT_TEMPERATURE"), defaultTemperature), "Temperature for the GPT model")
	model := pflag.StringP("model", "m", os.Getenv("SGPT_MODEL"), "GPT model to use")
	separator := pflag.StringP("separator", "s", os.Getenv("SGPT_SEPARATOR"), "Separator character for input")
	//debug := pflag.BoolP("debug", "d", parseBoolWithDefault(os.Getenv("SGPT_DEBUG"), false), "Enable debug output")

	// Set default values if not provided by any source
	if *separator == "" {
		*separator = "\n"
	}

	pflag.Parse()

	// Read the configuration file
	viper.SetConfigName(".sgpt")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.sgpt")
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Printf("Warning: Config file not found: %v", err)
		} else {
			log.Printf("Warning: Error reading config file: %v", err)
		}
	}

	// Bind command line flags to configuration
	viper.BindPFlag("apiKey", pflag.Lookup("api_key"))
	viper.BindPFlag("instruction", pflag.Lookup("i"))
	viper.BindPFlag("model", pflag.Lookup("m"))
	viper.BindPFlag("temperature", pflag.Lookup("t"))
	viper.BindPFlag("separator", pflag.Lookup("s"))
	viper.BindPFlag("debug", pflag.Lookup("d"))

	// Use values from viper, allowing for command line, env, or config file precedence
	*apiKey = viper.GetString("apiKey")
	*model = viper.GetString("model")
	*temperature = viper.GetFloat64("temperature")

	if *apiKey == "" {
		log.Fatal("API key is required but not provided through any configuration means.")
	}

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
