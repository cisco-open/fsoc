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

	"github.com/cisco-open/fsoc/cmd/config"
	openai "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

func init() {
	// Add the chat command to the root command
}

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Interact with ChatGPT",
	Long:  `This command allows you to interact with ChatGPT by sending messages.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get the API key from the environment variable or config file
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			cfg := config.GetCurrentContext()
			apiKey = cfg.OpenAIApiKey
		}

		// Initialize the OpenAI client
		client := openai.NewClient(apiKey)

		// Initialize the chatID and chatConversations map
		chatID := newChatID()
		chatConversations := initChatConversations()

		// Set the input message
		reader := bufio.NewReader(os.Stdin)

		fmt.Println("You are now in chat ID:", chatID)
		fmt.Println("Type 'exit' to stop the conversation or 'switch <chatID>' to switch to a different chat.")

		for {
			// Read input from the user
			fmt.Print("> ")
			inputMessage, _ := reader.ReadString('\n')
			inputMessage = strings.TrimSpace(inputMessage)

			if inputMessage == "exit" {
				break
			}

			if strings.HasPrefix(inputMessage, "switch ") {
				newID := strings.TrimSpace(strings.TrimPrefix(inputMessage, "switch"))
				if _, exists := chatConversations[newID]; exists {
					chatID = newID
					fmt.Println("You have switched to chat ID:", chatID)
				} else {
					fmt.Println("Chat ID not found.")
				}
				continue
			}

			// Add this block to handle commands within backticks
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

			// Add the input message to the conversation
			_ = continueChat(chatID, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: inputMessage,
			})

			// Send the message to ChatGPT
			resp, err := client.CreateChatCompletion(
				context.Background(),
				openai.ChatCompletionRequest{
					Model:    openai.GPT3Dot5Turbo,
					Messages: chatConversations[chatID].Messages,
				},
			)

			// Check for errors and print the response
			if err != nil {
				fmt.Printf("ChatCompletion error: %v\n", err)
				return
			}

			responseMessage := resp.Choices[0].Message.Content
			fmt.Printf("Assistant: %s\n", responseMessage)

			// Add the response message to the conversation
			_ = continueChat(chatID, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: responseMessage,
			})
		}
	},
}

// Chat is a struct representing a single chat conversation, which contains an array of ChatCompletionMessage.
type Chat struct {
	Messages []openai.ChatCompletionMessage
}

// chatConversations is a map that stores multiple chat conversations by chatID.
var chatConversations map[string]*Chat

// newChatID generates a new chat ID based on the length of the chatConversations map.
// The generated chat ID is a string representation of the incremented map length.
func newChatID() string {
	id := fmt.Sprintf("%d", len(chatConversations)+1)
	return id
}

// initChatConversations initializes the chatConversations map.
// This function should be called once at the beginning of the application.
func initChatConversations() map[string]*Chat {
	chatConversations = make(map[string]*Chat)
	return chatConversations
}

// continueChat appends a new message to an existing chat conversation identified by chatID.
// If the chatID doesn't exist in the chatConversations map, a new conversation is created.
// The function takes chatID and message as input parameters.
func continueChat(chatID string, message openai.ChatCompletionMessage) error {
	if _, exists := chatConversations[chatID]; !exists {
		chatConversations[chatID] = &Chat{Messages: []openai.ChatCompletionMessage{}}
	}

	chatConversations[chatID].Messages = append(chatConversations[chatID].Messages, message)
	return nil
}

// Add the runShellCommand function here
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

func NewChatCmd() *cobra.Command {
	return chatCmd
}
