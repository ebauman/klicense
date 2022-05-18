package key

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/ebauman/klicense/cert"
	"github.com/spf13/cobra"
	"os"
)

var writeFiles bool
var keyName string

func init() {
	generateCmd.Flags().StringVar(&keyName, "name", "license", "name of key")
	generateCmd.Flags().BoolVar(&writeFiles, "write-files", false, "whether to write files")
	Cmd.AddCommand(generateCmd)
}

var generateCmd = &cobra.Command{
	Use: "generate",
	Short: "generate",
	Aliases: []string{"gen", "g"},
	RunE: func(cmd *cobra.Command, args []string) error {
		key, err := cert.Generate()
		if err != nil {
			return err
		}

		keyPem := new(bytes.Buffer)
		err = pem.Encode(keyPem, &pem.Block{
			Type: "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		})
		if err != nil {
			return err
		}

		publicPem := new(bytes.Buffer)
		err = pem.Encode(publicPem, &pem.Block{
			Type: "CERTIFICATE",
			Bytes: x509.MarshalPKCS1PublicKey(&key.PublicKey),
		})

		if writeFiles {
			_ = os.WriteFile(fmt.Sprintf("%s.%s", keyName, "key"), keyPem.Bytes(), 0600)
			_ = os.WriteFile(fmt.Sprintf("%s.%s", keyName, "pem"), publicPem.Bytes(), 0600)
		} else {
			fmt.Println(keyPem)
			fmt.Println(publicPem)
		}

		return nil
	},
}