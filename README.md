# StreamGPT
StreamGPT (sgpt) is a command-line interface (CLI) tool to interact with OpenAI's API. It reads user input from standard input and sends it to the GPT model to generate a response based on the given instruction.  It writes these responses to standard output.  `sgpt` is intended for integration with toolchains.  It can operate on an input stream.

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

To install and use SGPT, follow these steps:

1. Ensure you have the Go programming language installed on your system. If not, follow the instructions at https://golang.org/doc/install.
2. Clone this repository to your local machine using `https://github.com/pdfinn/sgpt`.
3. Change to the `sgpt` directory and build the binary by running `go build`.
4. Make sure your OpenAI API key is available.

## Usage

For more information on OpenAI models see `https://platform.openai.com/docs/models/gpt-4`

```sh
sgpt -k <API_KEY> -i <INSTRUCTION> [-t TEMPERATURE] [-m MODEL] [-s SEPARATOR] [-d]
```

Here is a basic examples of how to use SGPT:

```sh
echo 'Hello GPT!' | sgpt -i 'you are a 1337 h4x0r who makes any input '1337'' -k <API_KEY>
```

```sh
cat sample.txt | sgpt -i 'You are an expert at analysing the sentiment of English statements. Analyze the sentiment and express it as an emoji.' -k <API_KEY>
```

```sh
echo 'If the coefficients of a quadratic equation are 1, 3, and -4, what are the roots of the equation?' | sgpt -i 'Answer the following question:' -k <API_KEY>
```

## Command-line flags and environment variables

| Flags        | Environment Variable	         | Config Key      | 	Description	                  | Default |
|--------------------|-------------------|-----------------|--------------------------------|---------|
| -k, --key	         | SGPT_API_KEY      | 	api_key	 | OpenAI API key                        | (none)  |
| -i, --instruction	 | SGPT_INSTRUCTION	 | instruction	    | Instruction for the GPT model  | 	(none) |
| -t, --temperature	 | SGPT_TEMPERATURE	 | temperature     | 	Temperature for the GPT model | 	0.5    |
| -m, --model	       | SGPT_MODEL	       | model           | GPT model to use	              | gpt-4   |
| -s, --separator    | 	SGPT_SEPARATOR   | 	separator      | 	Separator character for input | 	\n     |
| -d, --debug        | SGPT_DEBUG        | 	debug          | 	Enable debug output	          | false   |

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
