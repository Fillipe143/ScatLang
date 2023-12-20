package main

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"slices"
	"strings"
)

type Command struct {
	Name string
	Args int
	Run  func(args []int) string
}

type Token struct {
	Name   string
	Line   int
	Column int
}

var validKeywords = []string{"ski", "ba", "bop", "dop"}
var commands = []Command{
	{
		Name: "ski ba",
		Args: 1,
		Run: func(args []int) string {
			char := string(rune(args[0]))
			if char == "\n" {
				char = "0xA"
			} else {
				char = "'" + char + "'"
			}
			return fmt.Sprintf("mov byte [buff], %s\nmov eax, 0x4\nmov ebx, 0x1\nmov ecx, buff\nmov edx, 0x1\nint 0x80\n", char)
		},
	},
}

func main() {
	content := getContent()
	tokens := analyzeContent(content)
	data := parseTokens(tokens)
	generateExecutable(strings.TrimSuffix(os.Args[1], ".scl"), data)
}

func returnError(message string) {
	fmt.Printf("\033[0;31mError: %s\033[0m\n", message)
	os.Exit(1)
}

func generateExecutable(name string, data string) {
	binName := fmt.Sprintf(".tmp-%s.scl", name)

	err := os.WriteFile(binName+".asm", []byte(data), 0755)
	if err != nil {
		returnError(err.Error())
	}

	cmd := exec.Command("nasm", "-f", "elf64", binName+".asm")
	_, err = cmd.Output()
	if err != nil {
		returnError(err.Error())
	}

	cmd = exec.Command("ld", "-s", "-o", name, binName+".o")
	_, err = cmd.Output()
	if err != nil {
		returnError(err.Error())
	}

	cmd = exec.Command("rm", binName+".asm", binName+".o")
	_ = cmd.Run()
}

func getContent() string {
	args := os.Args
	if len(args) < 2 {
		returnError("File not specified")
	}

	filename := args[1]
	if !strings.HasSuffix(filename, ".scl") {
		filename += ".scl"
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		returnError(err.Error())
	}

	return string(content)
}

func analyzeContent(content string) []Token {
	var lineCounter int = 1
	var charCounter int = 0
	var word string = ""

	var tokens []Token = []Token{}

	for _, c := range content {
		charCounter++

		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			if word == "" {
				continue
			}

			if !slices.Contains(validKeywords, word) {
				returnError(fmt.Sprintf("Invalid keyword '%s' %d:%d", word, lineCounter, charCounter-len(word)))
			}

			tokens = append(tokens, Token{word, lineCounter, charCounter - len(word)})
			word = ""

			if c == '\n' {
				lineCounter++
				charCounter = 0
			}
		} else {
			word += string(c)
		}
	}

	return tokens
}

func parseTokens(tokens []Token) string {
	var result string = "section .data\nbuff db ' '\nsection .text\nglobal _start\n_start:\n"
	var commandName string
	var startCommand *Token

	var argument []Token
	var args []int

	for _, token := range tokens {
		if token.Name == "ski" || token.Name == "ba" {
			// Command
			if len(args) != 0 {
				result += runCommand(commandName, *startCommand, args)
				startCommand = nil
				commandName = ""
				args = []int{}
			}

			if len(argument) != 0 && len(argument) != 7 {
				startArgument := argument[0]
				returnError(fmt.Sprintf("Invalid argument %d:%d", startArgument.Line, startArgument.Column))
			}

			if startCommand == nil {
				startCommand = &token
			}

			commandName += token.Name + " "
		} else {
			// Argument
			if len(args) == 0 {
				if commandName == "" {
					returnError(fmt.Sprintf("Extra arguments %d:%d", token.Line, token.Column))
				}

				commandName = strings.TrimSpace(commandName)
			}

			argument = append(argument, token)

			if len(argument) == 7 {
				args = append(args, convertArgumentToInt(argument))
				argument = []Token{}
			}

		}
	}

	result += runCommand(commandName, *startCommand, args)
	return result + "mov eax, 0x1\nxor ebx, ebx\nint 0x80"
}

func runCommand(commandName string, startCommand Token, args []int) string {
	for _, command := range commands {
		if command.Name == commandName {
			if command.Args != len(args) {
				returnError(fmt.Sprintf("Invalid number of arguments for command '%s' %d:%d", commandName, startCommand.Line, startCommand.Column))
			}
			return command.Run(args)
		}
	}

	returnError(fmt.Sprintf("Invalid command '%s' %d:%d", commandName, startCommand.Line-1, startCommand.Column))
	return ""
}

func convertArgumentToInt(argument []Token) int {
	var result int = 0

	for i, token := range argument {
		if token.Name == "dop" {
			result += int(math.Pow(2, float64(len(argument)-i-1)))
		}
	}

	return result
}
