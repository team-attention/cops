package cobra

import (
	"github.com/spf13/cobra"
)

// NewRootCommand creates the root cobra command
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "code-rules",
		Short: "A CLI tool for managing code rules",
		Long:  "code-rules is a CLI tool that helps manage and enforce coding rules across your projects.",
	}

	return cmd
}
