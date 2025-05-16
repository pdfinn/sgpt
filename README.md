# StreamGPT

[![CI](https://github.com/pdfinn/sgpt/actions/workflows/ci.yml/badge.svg)](https://github.com/pdfinn/sgpt/actions/workflows/ci.yml)
[![Docker](https://github.com/pdfinn/sgpt/actions/workflows/docker.yml/badge.svg)](https://github.com/pdfinn/sgpt/actions/workflows/docker.yml)
[![Lint](https://github.com/pdfinn/sgpt/actions/workflows/lint.yml/badge.svg)](https://github.com/pdfinn/sgpt/actions/workflows/lint.yml)
[![Release](https://github.com/pdfinn/sgpt/actions/workflows/release.yml/badge.svg)](https://github.com/pdfinn/sgpt/actions/workflows/release.yml)

StreamGPT (SGPT) is a command-line interface (CLI) tool to interact with OpenAI's API, as well as Anthropic's Claude and Google's Gemini models. It reads user input from standard input and sends it to the selected AI model to generate a response based on the given instructions; it writes these responses to standard output. `sgpt` is intended for integration with toolchains and can operate on an input stream.

## Usage

```sh
sgpt -k <API_KEY> -i <INSTRUCTION> [-t TEMPERATURE] [-m MODEL] [-s SEPARATOR] [-p PROVIDER] [-d]
```

For more information on the available models, see:
- [OpenAI Models Documentation](https://platform.openai.com/docs/models)
- [Anthropic Models Documentation](https://docs.anthropic.com/claude/reference/selecting-a-model)
- [Google Gemini Documentation](https://ai.google.dev/models/gemini)

## Use Cases

StreamGPT is intended to merge [Unix design philosophy](https://en.wikipedia.org/wiki/Unix_philosophy) principles with the power of generative AI. It may be thought of as a general-purpose generative AI component that can be arbitrarily plugged into any text processing pipeline. SGPT helps make this convenient by allowing API keys and other parameters to be stored in a configuration file or environmental variables for easy application. A separator character (the default is a newline) may be specified to trigger application of the AI's instruction.

1. **Text Summarization:**

   Instruction: "Summarize the following text:"

   ```sh
   cat sample.txt | sgpt -k <API_KEY> -i "Summarize the following text:" -m "gpt-3.5-turbo"
   ```

2. **Text Translation:**

   Instruction: "Translate the following English text to 1337:"

   ```sh
   echo "Free Kevin!" | sgpt -k <API_KEY> -i "Translate the following English text to 1337:" -m "gpt-3.5-turbo"
   ```

3. **Sentiment Analysis:**

   Instruction: "You are an expert at analyzing the sentiment of English statements. Analyze the sentiment of each sample and express it as an emoji."

   ```sh
   cat samples.txt | sgpt -k <API_KEY> -i "Analyze the sentiment of each sample and express it as an emoji." -m "claude-v1" -p anthropic
   ```

4. **Code Generation:**

   Instruction: "Write a Python function to calculate the factorial of a given number:"

   ```sh
   echo "factorial" | sgpt -k <API_KEY> -i "Write a Python function to calculate the factorial of a given number:" -m "gemini-medium" -p google
   ```

5. **Image Analysis with Multimodal Models:**

   ```sh
   sgpt -k <API_KEY> -i "Describe what you see in this image:" -g path/to/image.jpg -m "gpt-4o" -p openai
   ```

## Features

- **Multi-provider Support:** Interact with OpenAI, Anthropic Claude, and Google Gemini models.
- **Stream Processing:** Read input from stdin, process it using the AI model, and output the response.
- **Configurable:** Configure the tool using command-line flags, environment variables, and a configuration file.
- **Adjustable Temperature:** Control randomness in the output.
- **Separator Character:** Specify a separator character to process input in chunks.
- **Debug Mode:** Enable debug output for troubleshooting.
- **Streaming Responses:** For supported models, stream tokens as they arrive for more responsive pipelines.
- **Multimodal Support:** Provide image and audio inputs to compatible models.

## Installation

### Download Pre-built Binaries

Download the latest version from the [releases](https://github.com/pdfinn/sgpt/releases) page.

#### macOS

```sh
# Using Homebrew
brew tap pdfinn/tap
brew install sgpt

# Manual installation
curl -L https://github.com/pdfinn/sgpt/releases/latest/download/sgpt-*-darwin-amd64.tar.gz | tar xz
sudo mv sgpt /usr/local/bin/
```

#### Linux

```sh
curl -L https://github.com/pdfinn/sgpt/releases/latest/download/sgpt-*-linux-amd64.tar.gz | tar xz
sudo mv sgpt /usr/local/bin/
```

#### Windows

Download the zip file from the [releases](https://github.com/pdfinn/sgpt/releases) page and extract it to a location in your PATH.

#### Docker

```sh
# Pull the latest image
docker pull ghcr.io/pdfinn/sgpt:latest

# Run with API key
docker run -it --rm \
  -e SGPT_API_KEY="your-api-key" \
  -e SGPT_PROVIDER="openai" \
  ghcr.io/pdfinn/sgpt -i "Translate to French:" "Hello world"

# Mount a config file
docker run -it --rm \
  -v $PWD/sgpt.yaml:/home/sgpt/.config/sgpt/sgpt.yaml \
  ghcr.io/pdfinn/sgpt -i "Summarize:" "Text to summarize"

# Process file input
cat input.txt | docker run -i --rm \
  -e SGPT_API_KEY="your-api-key" \
  ghcr.io/pdfinn/sgpt -i "Process this data:"
```

### Build from Source

1. **Install Go:**
   Ensure you have the Go programming language installed on your system (version 1.20 or later). If not, follow the instructions at [https://golang.org/doc/install](https://golang.org/doc/install).

2. **Clone the Repository:**

   ```sh
   git clone https://github.com/pdfinn/sgpt.git
   ```

3. **Build the Binary:**

   ```sh
   cd sgpt
   go build -o sgpt cmd/sgpt/main.go
   ```

4. **Set Up API Keys:**
   Make sure your API keys for the desired providers are available.

## Command-line Flags and Environment Variables

| Flags              | Environment Variable | Config Key    | Description                                | Default         |
|--------------------|----------------------|---------------|--------------------------------------------|-----------------|
| -k, --api_key      | SGPT_API_KEY         | api_key       | API key for the selected provider          | (none)          |
| -i, --instruction  | SGPT_INSTRUCTION     | instruction   | Instruction for the AI model               | (none)          |
| -t, --temperature  | SGPT_TEMPERATURE     | temperature   | Temperature for the AI model               | 0.5             |
| -m, --model        | SGPT_MODEL           | model         | AI model to use                            | Provider default|
| -s, --separator    | SGPT_SEPARATOR       | separator     | Separator character for input              | `\n`            |
| -p, --provider     | SGPT_PROVIDER        | provider      | Provider to use for API (openai, anthropic, google) | openai          |
| -d, --debug        | SGPT_DEBUG           | debug         | Enable debug output                        | false           |
| -g, --image        | SGPT_IMAGE           | image         | Path or URL to an image file               | (none)          |
| -a, --audio        | SGPT_AUDIO           | audio         | Path to an audio file                      | (none)          |

- **Note:** Command-line flags take precedence over environment variables, which take precedence over configuration file values.

## Configuration File

SGPT can be configured using a YAML configuration file. By default, SGPT looks for a file named `sgpt.yaml` in the current directory or `$HOME/`. This is especially useful for storing values that are not frequently changed, like the API key.

Example configuration file:

```yaml
api_key: your_api_key_here
instruction: "Translate the following English text to French:"
model: gpt-4
temperature: 0.5
separator: "\n"
debug: false
provider: openai
```

## Architecture

StreamGPT is designed with the following principles in mind:

- **Modularity:** Each component does one thing well and can be tested independently.
- **Interfaces:** The provider interface allows for easy addition of new AI backends.
- **Configuration:** All settings are centralized and validated before use.
- **Streaming:** Responses are streamed when supported for maximum responsiveness.
- **Security:** Debug logging redacts sensitive information.

The main components include:

- **Config:** Manages configuration from flags, environment, and files.
- **Provider:** Defines the interface that all AI backends implement.
- **Transport:** Handles HTTP requests with connection pooling and timeouts.
- **Command-line Interface:** Minimal parsing and wiring of components.

## Development

To run the tests:

```sh
go test ./...
```

## Contributing

Contributions are welcome! Here's how the development workflow is set up:

### GitHub Actions

This project uses GitHub Actions for CI/CD:

- **CI**: Runs tests and builds binaries for different platforms on every push and PR
- **Lint**: Runs `golangci-lint` to check code quality on every push and PR
- **Docker**: Builds and publishes a Docker image to GitHub Container Registry
- **Release**: Creates a release with binaries and Homebrew formula when a new tag is pushed

### Creating a Release

To create a new release:

1. Update code and tests
2. Run tests locally: `go test ./...`
3. Commit changes and push to main
4. Tag a new version: `git tag v1.2.3`
5. Push the tag: `git push origin v1.2.3`

GitHub Actions will automatically build and publish the release artifacts.

## License

This project is released under the MIT License. See the [LICENSE](LICENSE) file for more information.
