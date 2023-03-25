package openai

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/cisco-open/fsoc/cmd/config"
	openai "github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

type ChatConversation struct {
	ID       string
	Messages []openai.ChatCompletionMessage
}

var chatConversations = make(map[string]*ChatConversation)

func generateUniqueID() string {
	return fmt.Sprintf("%d", len(chatConversations)+1)
}

func startNewChat() string {
	chatID := generateUniqueID()
	chatConversations[chatID] = &ChatConversation{
		ID:       chatID,
		Messages: []openai.ChatCompletionMessage{},
	}
	return chatID
}

func continueChat(chatID string, message openai.ChatCompletionMessage) error {
	conversation, ok := chatConversations[chatID]
	if !ok {
		return errors.New("conversation not found")
	}
	conversation.Messages = append(conversation.Messages, message)
	return nil
}

func switchChat(chatID string) error {
	_, ok := chatConversations[chatID]
	if !ok {
		return errors.New("conversation not found")
	}
	return nil
}

var chatCmd = &cobra.Command{
	Use:   "chat [start|continue <chatID>|switch <chatID>] <message>",
	Short: "Interact with ChatGPT",
	Long:  `This command allows you to interact with ChatGPT by sending messages.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		action := args[0]
		var inputMessage string
		var chatID string
		var err error

		switch action {
		case "start":
			inputMessage = strings.Join(args[1:], " ")
			chatID = startNewChat()
		case "continue":
			if len(args) < 3 {
				fmt.Println("Please provide a chat ID and a message.")
				return
			}
			chatID = args[1]
			inputMessage = strings.Join(args[2:], " ")
			err = continueChat(chatID, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: inputMessage,
			})
			if err != nil {
				fmt.Println(err)
				return
			}
		case "switch":
			if len(args) < 2 {
				fmt.Println("Please provide a chat ID.")
				return
			}
			chatID = args[1]
			err = switchChat(chatID)
			if err != nil {
				fmt.Println(err)
				return
			}
			return
		default:
			fmt.Println("Invalid action. Use 'start', 'continue', or 'switch'.")
			return
		}

		// Get the API key from the environment variable or config file
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			cfg := config.GetCurrentContext()
			apiKey = cfg.OpenAIApiKey
		}

		// Initialize the OpenAI client
		client := openai.NewClient(apiKey)

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
		fmt.Println("Chat ID:", chatID)
		fmt.Println("Response:", responseMessage)

		// Add the response message to the conversation
		_ = continueChat(chatID, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: responseMessage,
		})
	},
}

func NewChatCmd() *cobra.Command {
	return chatCmd
}
