package kubernetes

import (
	"fmt"
	"k8s.io/apimachinery/pkg/types"
)

type NamespacedName struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

func (n *NamespacedName) String() string {
	return fmt.Sprintf("%s/%s", n.Namespace, n.Name)
}

func (n *NamespacedName) ToK8sNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      n.Name,
		Namespace: n.Namespace,
	}
}
