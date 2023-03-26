package openai

import (
	"fmt"

	"github.com/sashabaranov/go-openai"
)

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
