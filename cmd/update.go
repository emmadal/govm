package cmd

import (
	"fmt"
	"github.com/emmadal/govm/internal"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "update govm to the latest version",
	Example: strings.Join([]string{"$ govm update"}, "\n"),
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("expect no arguments")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Stop cleanly on Ctrl-C: the download is aborted and cleaned up.
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return internal.UpdateGovm(ctx)
	},
}
