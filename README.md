# StreamGPT
StreamGPT (sgpt) is a command-line interface (CLI) tool to interact with OpenAI's API. It reads user input from standard input and sends it to the GPT model to generate a response based on the given instruction.  It writes these responses to standard output.  `sgpt` is intended for integration with toolchains.  It can operate on an input stream.

## Usage

```sh
sgpt -k <API_KEY> -i <INSTRUCTION> [-t TEMPERATURE] [-m MODEL] [-s SEPARATOR] [-d]
```
For more information on OpenAI models see `https://platform.openai.com/docs/models/gpt-4`


## Use cases

StreamGPT is intended to merge [Unix design philosophy](https://en.wikipedia.org/wiki/Unix_philosophy) principles with the power of generative AI.  It may be thought of as a general-purpose generative AI component that can be arbitrarily plugged into any text processing operation.  SGPT helps make this convenient by allowyng API keys and other settings to be stored in a configuration file.  A seperator character (the default is a new-line) may be specified to trigger application of the AI's instruction.

1) Text summarization:

   Instruction: "Summarize the following text:"

    ```sh
   cat sample.txt | sgpt --api_key YOUR_API_KEY --instruction "Summarize the following text:" --model "gpt-3.5-turbo"
   ```

2) Text translation:

   Instruction: "Translate the following English text to 1337:"
   Example usage:

    ```sh
   echo "Free Kevin!" | sgpt -i "you are a 1337 h4x0r who translates any input to '1337'" -k <API_KEY>
   ```

3) Sentiment analysis:
   Instruction: "You are an expert at analysing the sentiment of English statements. Analyze the sentiment of each sample and express it as an emoji."
   Example usage:

    ```sh
   cat sample.txt | sgpt -i "You are an expert at analysing the sentiment of English statements. Analyze the sentiment of each sample and express it as an emoji." -k <API_KEY>
   ```

4) Code generation:
Instruction: "Write a Python function to calculate the factorial of a given number:"
Example usage:

    ```sh
   echo "factorial" | sgpt --api_key YOUR_API_KEY --instruction "Write a Python function to calculate the factorial of a given number:" --model "gpt-3.5-turbo"
    ```

## Features

- Read input from stdin, process it using the GPT model, and output the response
- Configure the tool using command-line flags, environment variables, and a configuration file
- Support for different GPT models
- Adjustable temperature for controlling randomness in the output
- Supported models:
    - GPT-4:
        - `gpt-4`
        - `gpt-4-0314`
        - `gpt-4-32k`
        - `gpt-4-32k-0314`
    - GPT-3:
        - `gpt-3.5-turbo`
        - `gpt-3.5-turbo-0301`
        - `text-davinci-003`
        - `text-davinci-002`
        - `text-curie-001`
        - `text-babbage-001`
        - `text-ada-001`

## Installation

To install SGPT download the version from the `build` directory in a location such as `$HOME/bin/`

To build SGPT from source, follow these steps:

1. Ensure you have the Go programming language installed on your system. If not, follow the instructions at https://golang.org/doc/install.
2. Clone this repository to your local machine using `https://github.com/pdfinn/sgpt`.
3. Change to the `sgpt` directory and build the binary by running `go build`.
4. Make sure your OpenAI API key is available.


## Command-line flags and environment variables

| Flags              | Environment Variable	         | Config Key      | 	Description	                  | Default       |
|--------------------|-------------------|-----------------|--------------------------------|---------------|
| -k, --api_key	     | SGPT_API_KEY      | 	api_key	 | OpenAI API key                        | (none)        |
| -i, --instruction	 | SGPT_INSTRUCTION	 | instruction	    | Instruction for the GPT model  | 	(none)       |
| -t, --temperature	 | SGPT_TEMPERATURE	 | temperature     | 	Temperature for the GPT model | 	0.5          |
| -m, --model	       | SGPT_MODEL	       | model           | GPT model to use	              | gpt-3.5-turbo |
| -s, --separator    | 	SGPT_SEPARATOR   | 	separator      | 	Separator character for input | 	\n           |
| -d, --debug        | SGPT_DEBUG        | 	debug          | 	Enable debug output	          | false         |

- Note: Command line flags take precedence over environment variables.

## Configuration File
SGPT can be configured using a YAML configuration file. By default, SGPT looks for a file named `sgpt.yaml` in the current directory or `$HOME/.sgpt`.  This is especially useful for storing values that are not frequently changed, like the API key

Example configuration file:

```
api_key: your_api_key_here
instruction: "Translate the following English text to French:"
model: gpt-4
temperature: 0.5
separator: "\n"
debug: false
```

## Order of Preference
The order of preference for configuration values is as follows:

1. Command-line flags
2. Environment variables
3. Configuration file

When a value is set using multiple methods, the method with the highest precedence will be used. For example, if a value is set using both a command-line flag and an environment variable, the value from the command-line flag will be used.

## License

This project is released under the MIT License. See the LICENSE file for more information.
