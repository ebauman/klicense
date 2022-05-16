package license

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"k8s.io/apimachinery/pkg/util/json"
	"strings"
	"time"
)

var publicKeys = make([]*rsa.PublicKey, 0)

func init() {
	for _, key := range keys {
		pBlock, _ := pem.Decode([]byte(key))

		if pBlock == nil {
			continue
		}

		pubkey, err := x509.ParsePKCS1PublicKey(pBlock.Bytes)
		if err != nil {
			continue
		}

		publicKeys = append(publicKeys, pubkey)
	}
}

type License struct {
	Id        string            `json:"id"`
	Licensee  string            `json:"licensee"`
	Metadata  map[string]string `json:"metadata"`
	Grants    map[string]int    `json:"grants"`
	NotBefore time.Time         `json:"notBefore"`
	NotAfter  time.Time         `json:"notAfter"`
}

func Validate(licenseBytes []byte) (*License, error) {
	if len(licenseBytes) == 0 {
		return nil, fmt.Errorf("invalid license")
	}

	licenseSlice := strings.Split(string(licenseBytes), ".")
	// first part of the slice is the json content
	// second part is the sign of the hash
	hash := sha256.New()
	_, err := hash.Write([]byte(licenseSlice[0]))
	if err != nil {
		return nil, fmt.Errorf("invalid license")
	}
	hashSum := hash.Sum(nil)

	signature, err := base64.StdEncoding.DecodeString(licenseSlice[1])
	if err != nil {
		return nil, fmt.Errorf("invalid license")
	}

	var valid = false
	for _, key := range publicKeys {
		err = rsa.VerifyPSS(key, crypto.SHA256, hashSum, signature, nil)
		if err == nil {
			valid = true
			break
		}
	}

	if !valid {
		return nil, fmt.Errorf("invalid license")
	}

	// decode base64 into json, and turn it into a license
	licenseJson, err := base64.StdEncoding.DecodeString(licenseSlice[0])
	if err != nil {
		return nil, fmt.Errorf("invalid license")
	}

	var license = License{}
	if err = json.Unmarshal(licenseJson, &license); err != nil {
		return nil, fmt.Errorf("invalid license")
	}

	return &license, nil
}
