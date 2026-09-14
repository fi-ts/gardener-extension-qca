package qca

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// QCAConfig defines the configuration for the qualys controller.
type QCAConfig struct {
	metav1.TypeMeta
	TenantId string
	// Proxy is an optional proxy that is used for this activation.
	// If not set, the proxy of the tenant config is used. If this is also
	// not set, the globally configured proxy is used.
	// +optional
	Proxy string
}
