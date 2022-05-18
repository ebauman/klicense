package license

import (
	license2 "github.com/ebauman/klicense/license"
	"github.com/spf13/cobra"
)

var license = license2.License {
	Grants:   map[string]int{},
	Metadata: map[string]string{},
}

var metadataSlice []string
var grantSlice []string
var keyFilePath string

var notBefore string
var notAfter string

var Cmd = &cobra.Command{
	Use: "license",
	Short: "operations on licenses",
}