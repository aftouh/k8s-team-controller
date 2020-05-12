package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

//Team defines team resource structure
type Team struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamSpec   `json:"spec"`
	Status TeamStatus `json:"status"`
}

// TeamSpec is the spec for a team resource
type TeamSpec struct {
	Name              string                   `json:"name"`
	Environment       string                   `json:"environment"`
	Description       string                   `json:"description"`
	ResourceQuotaSpec corev1.ResourceQuotaSpec `json:"resourceQuota"`
}

// TeamStatus is the status for a Team resource
type TeamStatus struct {
	Namespace     string `json:"namespace"`
	ResourceQuota string `json:"resourcequota"`
}

// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TeamList is a list of Team resources
type TeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Team `json:"items"`
}
