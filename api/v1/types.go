package v1

import (
	"github.com/ebauman/klicense/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	GrantStatusFree    GrantStatus = "Free"
	GrantStatusPending GrantStatus = "Pending"
	GrantStatusInUse   GrantStatus = "InUse"

	UsageRequestStatusDiscover UsageRequestStatus = "Discover"
	UsageRequestStatusOffer        UsageRequestStatus = "Offer"
	UsageRequestStatusAcknowledged UsageRequestStatus = "Acknowledged"
)

type GrantStatus string

type Grant struct {
	Id            string                    `json:"id"`
	Amount        int                       `json:"amount"`
	Type          string                    `json:"type"`
	NotBefore     metav1.Time               `json:"notBefore"`
	NotAfter      metav1.Time               `json:"notAfter"`
	LicenseSecret kubernetes.NamespacedName `json:"licenseSecret"`
	Status        GrantStatus               `json:"grantStatus"`
	Request       kubernetes.NamespacedName `json:"request"`
}

type EntitlementStatus struct {
	Grants map[string]Grant `json:"grants"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Entitlement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status EntitlementStatus `json:"status,omitempty"`
}


type UsageRequestStatus string

type RequestSpec struct {
	Entitlement kubernetes.NamespacedName `json:"entitlement"`
	Type        string                    `json:"type"`
	Amount      int                       `json:"amount"`
}

type RequestStatus struct {
	Status UsageRequestStatus `json:"status"`
	Grant  string             `json:"grant"`
	Message string `json:"message"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Request struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RequestSpec   `json:"spec,omitempty"`
	Status RequestStatus `json:"status,omitempty"`
}