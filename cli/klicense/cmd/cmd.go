package cmd

import (
	"fmt"
	"github.com/ebauman/klicense/cli/klicense/cmd/key"
	"github.com/ebauman/klicense/cli/klicense/cmd/license"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	rootCmd.AddCommand(license.Cmd)
	rootCmd.AddCommand(key.Cmd)
}

var rootCmd = &cobra.Command {
	Use: "lc",
	Short: "manage licenses",
	Long: "manages keys and licenses which can be used for software entitlements",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}