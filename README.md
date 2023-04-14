# StreamGPT
StreamGPT (sgpt) is a command-line tool to interact with OpenAI's API. It reads user input from standard input and sends it to the GPT model to generate a response based on the given instruction.  It writes these responses to standard output.  `sgpt` is intended for integration with toolchains.  It can operate on an input stream.

## Features

- Interact with OpenAI's GPT-4 and GPT-3 models
- Customizable model prompt and temperature
- Debug mode for detailed API response information
- Support for environmental variables
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

## Supported models

For more information on OpenAI models see `https://platform.openai.com/docs/models/gpt-4`

## Usage

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

## Command-line flags
- `-k` (required): Your OpenAI API key
- `-i` (required): The instruction for the GPT model
- `-t`: The temperature for the GPT model (default: 0.5)
- `-m`: The GPT model to use (default: "`gpt-4`")
- `-s`: Separator character for input (default: `\n`)
- `-d`: Enable debug output (default: false)

## Command-line flags and environment variables

- `-k` (required): Your OpenAI API key. Can also be set with the `SGPT_API_KEY` environment variable.
- `-i` (required): The instruction for the GPT model. Can also be set with the `SGPT_INSTRUCTION` environment variable.
- `-t:`  The temperature for the GPT model (default: 0.5). Can also be set with the `SGPT_TEMPERATURE` environment variable.
- `-m:`  The GPT model to use (default: "`gpt-4`"). Can also be set with the `SGPT_MODEL` environment variable.
- `-s:`  Separator character for input (default: `\n`). Can also be set with the `SGPT_SEPARATOR` environment variable.
- `-d:`  Enable debug output (default: `false`). Can also be set with the `SGPT_DEBUG` environment variable.

- Note: Command line flags take precedence over environment variables.

## License

This project is released under the MIT License. See the LICENSE file for more information.
