package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

func Generate() (*rsa.PrivateKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	return key, nil
}

func LoadKey(path string) (*rsa.PrivateKey, error) {
	keyPem, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return decodeKey(keyPem)
}

func decodeKey(data []byte) (*rsa.PrivateKey, error) {
	pBlock, _ := pem.Decode(data)

	if pBlock == nil {
		return nil, fmt.Errorf("error decoding pem data from file")
	}

	if pBlock.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("invalid key, not rsa private key")
	}

	key, err := x509.ParsePKCS1PrivateKey(pBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing rsa private key: %v", err)
	}

	return key, nil
}