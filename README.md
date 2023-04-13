# SGPT
StreamGPT (sgpt) is a command-line tool to interact with OpenAI's API. It reads user input from standard input and sends it to the GPT model to generate a response based on the given instruction.  It writes these responses to standard output.  `sgpt` is intended for intended for integration with toolchains.  It can operate on an input stream.

## Features

- Interact with OpenAI's GPT-4 model
- Customizable model prompt and temperature
- Debug mode for detailed API response information

## Installation

To install and use SGPT, follow these steps:

1. Ensure you have the Go programming language installed on your system. If not, follow the instructions at https://golang.org/doc/install.
2. Clone this repository to your local machine using `https://github.com/pdfinn/sgpt`.
3. Change to the `sgpt` directory and build the binary by running `go build`.
4. Make sure your OpenAI API key is available.

## Usage

```sh
sgpt -k <API_KEY> -i <INSTRUCTION> [-t TEMPERATURE] [-m MODEL] [-s SEPARATOR] [-d]
```

Here is a basic examples of how to use SGPT:

```sh
echo 'Hello GPT!' | ./sgpt -i 'you are a 733t h4x0r who makes any input 733t' -k YOUR_API_KEY
```

```sh
cat sentiment.txt | ./sgpt -i 'You are an expert at analysing the sentiment of English statements. Analyze the sentiment and express it as an emoji.' -k YOUR_API_KEY
```

```sh
echo "What is the capital of France?" | ./sgpt -k <API_KEY> -i "Answer the following question:"
```

## Command-line flags
- `-k` (required): Your OpenAI API key
- `-i` (required): The instruction for the GPT model
- `-t`: The temperature for the GPT model (default: 0.5)
- `-m`: The GPT model to use (default: "`gpt-4`")
- `-d`: Enable debug output (default: false)
- `-s`: Separator character for input (default: `\n`).

## License

This project is released under the MIT License. See the LICENSE file for more information.
