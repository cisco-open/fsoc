package openai

import (
	"github.com/spf13/cobra"
)

// NewOpenaiCmd creates a new 'openai' command
func NewSubCmd() *cobra.Command {
	openaiCmd := &cobra.Command{
		Use:   "openai",
		Short: "Interact with OpenAI",
		Long: `This command allows you to interact with OpenAI services.

Usage:
fsoc openai`,
		TraverseChildren: true,
	}

	// Add the 'chat' subcommand to the 'openai' command
	openaiCmd.AddCommand(NewChatCmd())

	return openaiCmd
}
