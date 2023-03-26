package openai

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// readUserInput reads the input from the user.
func readUserInput(reader *bufio.Reader) string {
	fmt.Print("> ")
	inputMessage, _ := reader.ReadString('\n')
	inputMessage = strings.TrimSpace(inputMessage)
	return inputMessage
}

// exitOrSwitchCommand handles exit and switch commands, returning true if either is executed.
func exitOrSwitchCommand(inputMessage string, chatID *string) bool {
	if inputMessage == "exit" {
		os.Exit(0)
	}

	if strings.HasPrefix(inputMessage, "switch ") {
		newID := strings.TrimSpace(strings.TrimPrefix(inputMessage, "switch"))
		if _, exists := chatConversations[newID]; exists {
			*chatID = newID
			fmt.Println("You have switched to chat ID:", *chatID)
		} else {
			fmt.Println("Chat ID not found.")
		}
		return true
	}

	return false
}

// replaceShellCommands replaces shell commands within backticks with their output.
func replaceShellCommands(inputMessage string) string {
	regex := regexp.MustCompile("`([^`]*)`")
	inputMessage = regex.ReplaceAllStringFunc(inputMessage, func(matched string) string {
		shellCommand := strings.Trim(matched, "`")
		cmdOutput, err := runShellCommand(shellCommand)
		if err != nil {
			fmt.Printf("Error executing command '%s': %v\n", shellCommand, err)
			return "<error executing command>"
		}
		return cmdOutput
	})
	return inputMessage
}

// processUserInput adds the input message to the conversation and sends the message to ChatGPT.
func processUserInput(client *openai.Client, model string, chatID string, inputMessage string) (string, error) {
	_ = continueChat(chatID, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: inputMessage,
	})

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    model,
			Messages: chatConversations[chatID].Messages,
		},
	)

	if err != nil {
		return "", err
	}

	responseMessage := resp.Choices[0].Message.Content
	return responseMessage, nil
}

// runShellCommand executes a shell command and returns its output.
func runShellCommand(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return out.String(), nil
}
