package license

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
)

func ValidateSecret(secret *corev1.Secret) (*License, error) {
	licenseData, ok := secret.Data["license"]
	if !ok {
		return nil, fmt.Errorf("secret does not contain license field")
	}

	return Validate(licenseData)
}
