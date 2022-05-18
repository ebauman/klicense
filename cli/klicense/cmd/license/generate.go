package license

import (
	"fmt"
	"github.com/ebauman/klicense/cert"
	license2 "github.com/ebauman/klicense/license"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"time"
)

func init() {
	generateCmd.Flags().StringVar(&license.Licensee, "licensee", "", "name/id of licensee")
	generateCmd.Flags().StringSliceVar(&metadataSlice, "metadata", []string{}, "metadata")
	generateCmd.Flags().StringSliceVar(&grantSlice, "grant", []string{}, "grant")
	generateCmd.Flags().StringVar(&keyFilePath, "key", "", "key")
	generateCmd.Flags().StringVar(&notBefore, "not-before", time.Now().Format("2006-01-02"), "license not valid before this date (yyyy-mm-dd)")
	generateCmd.Flags().StringVar(&notAfter, "not-after", "", "license not valid after this date (yyyy-mm-dd)")

	for _, v := range []string{"licensee", "grant", "key", "not-after"} {
		_ = generateCmd.MarkFlagRequired(v)
	}

	Cmd.AddCommand(generateCmd)
}

var generateCmd = &cobra.Command {
	Use: "generate",
	Aliases: []string{"gen", "g"},
	Short: "generate a license key",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := license2.FlagsToMetadata(metadataSlice, &license); err != nil {
			return err
		}
		if err := license2.FlagsToGrants(grantSlice, &license); err != nil {
			return err
		}
		if err := license2.FlagToNotAfter(notAfter, &license); err != nil {
			return err
		}
		if err := license2.FlagToNotBefore(notBefore, &license); err != nil {
			return err
		}

		key, err := cert.LoadKey(keyFilePath)
		if err != nil {
			return err
		}

		license.Id = uuid.NewString()
		license, err := license2.Generate(key, license)
		if err != nil {
			return err
		}

		fmt.Println(license)
		return nil
	},
}