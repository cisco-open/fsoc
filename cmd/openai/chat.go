// Copyright 2023 Your Company, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openai

import (
	"context"
	"fmt"
	"os"

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

		// Set the input message
		inputMessage := "your input message"

		// Send the message to ChatGPT
		resp, err := client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model: openai.GPT3Dot5Turbo,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleUser,
						Content: inputMessage,
					},
				},
			},
		)

		// Check for errors and print the response
		if err != nil {
			fmt.Printf("ChatCompletion error: %v\n", err)
			return
		}

		fmt.Println(resp.Choices[0].Message.Content)
	},
}

func NewChatCmd() *cobra.Command {
	return chatCmd
}
