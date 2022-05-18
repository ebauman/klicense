package license

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"k8s.io/apimachinery/pkg/util/json"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	grantString = `^(([a-zA-Z]{1})|([a-zA-Z]{1}[a-zA-Z]{1})|([a-zA-Z]{1}[0-9]{1})|([0-9]{1}[a-zA-Z]{1})|([a-zA-Z0-9][a-zA-Z0-9-_]{1,61}[a-zA-Z0-9]))\.([a-zA-Z]{2,6}|[a-zA-Z0-9-]{2,30}\.[a-zA-Z]{2,3})\/[a-zA-Z0-9]{1,}=[0-9]{1,}$`
	entitlementName = `(([a-zA-Z]{1})|([a-zA-Z]{1}[a-zA-Z]{1})|([a-zA-Z]{1}[0-9]{1})|([0-9]{1}[a-zA-Z]{1})|([a-zA-Z0-9][a-zA-Z0-9-_]{1,61}[a-zA-Z0-9]))\.([a-zA-Z]{2,6}|[a-zA-Z0-9-]{2,30}\.[a-zA-Z]{2,3})`
)

var GrantStringRegexp *regexp.Regexp
var EntitlementNameRegexp *regexp.Regexp

func init() {
	GrantStringRegexp = regexp.MustCompile(grantString)
	EntitlementNameRegexp = regexp.MustCompile(entitlementName)
}

type License struct {
	Id        string            `json:"id"`
	Licensee  string            `json:"licensee"`
	Metadata  map[string]string `json:"metadata"`
	Grants    map[string]int    `json:"grants"`
	NotBefore time.Time         `json:"notBefore"`
	NotAfter  time.Time         `json:"notAfter"`
}

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

func FlagsToMetadata(flags []string, license *License) error {
	return flagsTo(flags, license, "metadata")
}

func FlagsToGrants(flags []string, license *License) error {
	return flagsTo(flags, license, "grant")
}

func FlagToNotAfter(flag string, license *License) error {
	return flagToTime(flag, license, "not-after")
}

func FlagToNotBefore(flag string, license *License) error {
	return flagToTime(flag, license, "not-before")
}

func flagToTime(flag string, license *License, kind string) error {
	t, err := time.Parse("2006-01-02", flag)
	if err != nil {
		return fmt.Errorf("invalid date %s: %s", flag, err)
	}

	if kind == "not-before" {
		license.NotBefore = t
	} else {
		license.NotAfter = t
	}

	return nil
}

func flagsTo(flags []string, license *License, location string) error {
	for _, v := range flags {
		// check if the flag matches the regex
		if !GrantStringRegexp.Match([]byte(v)) {
			return fmt.Errorf("grant string %s is not of the format sub.doma.in/unit=123", v)
		}

		split := strings.Split(v, "=")
		if len(split) <2 {
			return fmt.Errorf("invalid %s: %s", location, v)
		}

		num, err := strconv.Atoi(split[1])
		if err != nil {
			return fmt.Errorf("invalid %s: %s is not a number", location, split[1])
		}

		if location == "metadata" {
			license.Metadata[split[0]] = split[1]
		} else {
			license.Grants[split[0]] = num
		}
	}
	return nil
}

func Generate(key *rsa.PrivateKey, license License) (string, error) {
	licenseJson, err := json.Marshal(license)
	if err != nil {
		return "", err
	}

	base64License := encode(licenseJson)

	hash := sha256.New()
	_, err = hash.Write(base64License)
	if err != nil {
		return "", err
	}
	hashSum := hash.Sum(nil)

	signature, err := rsa.SignPSS(rand.Reader, key, crypto.SHA256, hashSum, nil)
	if err != nil {
		return "", err
	}

	base64Signature := encode(signature)

	return fmt.Sprintf("%s.%s", base64License, base64Signature), nil
}

func encode(in []byte) []byte {
	var buf = &bytes.Buffer{}
	encoder := base64.NewEncoder(base64.StdEncoding, buf)

	encoder.Write(in)

	encoder.Close()

	return buf.Bytes()
}
