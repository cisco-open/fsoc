package chat

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/llms/openai"
)

func runChatCmd(cmd *cobra.Command, args []string) {
	ctx := context.Background() // create a context
	llm, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	completion, err := llm.Call(ctx, args[0]) // Pass the context and user's input text to the GPT-4 model
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(completion)
}

// NewSubCmd creates a new 'chat' command
func NewSubCmd() *cobra.Command {
	chatCmd := &cobra.Command{
		Use:   "chat",
		Short: "Run FSO Platform commands in a natural language",
		Long: `This command allows you to interact with FSO Platform using NLP.

Usage:
fsoc chat "message"`,
		TraverseChildren: true,
		Args:             cobra.ExactArgs(1), // Expect exactly one argument: the input text
		Run:              runChatCmd,
	}

	return chatCmd
}
