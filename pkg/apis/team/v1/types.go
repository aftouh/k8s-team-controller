package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

//Team defines team resource structure
type Team struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec TeamSpec `json:"spec"`
	//Status TeamStatus `json:"status"`
}

// TeamSpec is the spec for a Boo resource
type TeamSpec struct {
	Name string `json:"name"`
	Size *int32 `json:"size"`
}

// TeamStatus is the status for a Team resource
// type TeamStatus struct {
// 	AvailableReplicas int32 `json:"availableReplicas"`
// }

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TeamList is a list of Team resources
type TeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Team `json:"items"`
}
