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

// Initialize chat command and add it to the root command
func init() {
	chatCmd.Flags().String("model", "", "Specify the model to use for the chat (default: "+openai.GPT3Dot5Turbo+")")
}

// Chat is a struct representing a single chat conversation, which contains an array of ChatCompletionMessage.
type Chat struct {
	Messages []openai.ChatCompletionMessage
}

// chatCmd defines the chat command for interacting with ChatGPT.
var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Interact with ChatGPT",
	Long: `Interact with ChatGPT by sending messages. You can send text input and receive AI-generated responses.
	Type 'exit' to stop the conversation or 'switch <chatID>' to switch to a different chat.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get the API key from the environment variable or config file
		apiKey := getAPIKey()

		// Get the selected model from the flag
		model := getModel(cmd)

		// Initialize the OpenAI client
		client := openai.NewClient(apiKey)

		// Initialize the chatID and chatConversations map
		chatID := newChatID()
		_ = initChatConversations()

		// Display the initial chat session information
		displayChatSessionInfo(chatID)

		// Set the input message
		reader := bufio.NewReader(os.Stdin)

		for {
			// Read input from the user
			inputMessage := readUserInput(reader)

			// Handle exit and switch commands
			if exitOrSwitchCommand(inputMessage, &chatID) {
				continue
			}

			// Replace shell commands within backticks with their output
			inputMessage = replaceShellCommands(inputMessage)

			// Add the input message to the conversation and send the message to ChatGPT
			responseMessage, err := processUserInput(client, model, chatID, inputMessage)

			// Check for errors and print the response
			if err != nil {
				fmt.Printf("ChatCompletion error: %v\n", err)
				return
			}

			fmt.Printf("Assistant: %s\n", responseMessage)

			// Add the response message to the conversation
			_ = continueChat(chatID, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: responseMessage,
			})
		}
	},
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

// runShellCommand executes the given shell command and returns its output as a string.
// In case of an error, it returns an empty string and the error.
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

// getAPIKey retrieves the API key from the environment variable or config file.
func getAPIKey() string {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		cfg := config.GetCurrentContext()
		apiKey = cfg.OpenAIApiKey
	}
	return apiKey
}

// getModel retrieves the selected model from the flag or uses the default.
func getModel(cmd *cobra.Command) string {
	model, _ := cmd.Flags().GetString("model")
	if model == "" {
		model = openai.GPT3Dot5Turbo
	}
	return model
}

// displayChatSessionInfo displays the initial chat session information.
func displayChatSessionInfo(chatID string) {
	fmt.Println("You are now in chat ID:", chatID)
	fmt.Println("Type 'exit' to stop the conversation or 'switch <chatID>' to switch to a different chat.")
}

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

// NewChatCmd returns the chatCmd to be used by the main application.
func NewChatCmd() *cobra.Command {
	return chatCmd
}
