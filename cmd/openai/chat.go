package openai

import (
	"bufio"
	"fmt"
	"os"

	"github.com/cisco-open/fsoc/cmd/config"
	openai "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

// Initialize chat command and add it to the root command
func init() {
	chatCmd.Flags().String("model", "", "Specify the model to use for the chat (default: "+openai.GPT3Dot5Turbo+")")
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

// NewChatCmd returns the chatCmd to be used by the main application.
func NewChatCmd() *cobra.Command {
	return chatCmd
}
