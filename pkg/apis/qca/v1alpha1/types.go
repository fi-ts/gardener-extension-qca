package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// QCAConfig defines the configuration for the qualys controller.
type QCAConfig struct {
	metav1.TypeMeta `json:",inline"`
	TenantId        string `json:"tenantId"`
}
