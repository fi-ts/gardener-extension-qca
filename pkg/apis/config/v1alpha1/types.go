package v1alpha1

import (
	healthcheckconfigv1alpha1 "github.com/gardener/gardener/extensions/pkg/apis/config/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ControllerConfiguration configuration resource
type ControllerConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// HealthCheckConfig is the config for the health check controller
	// +optional
	HealthCheckConfig *healthcheckconfigv1alpha1.HealthCheckConfig `json:"healthCheckConfig,omitempty"`

	ActivationId  string        `json:"activationId"`
	Server        string        `json:"server"`
	Proxy         string        `json:"proxy"`
	TenantConfigs TenantConfigs `json:"tenantConfigs,omitempty"`
}

type TenantConfig struct {
	TenantId   string `json:"tenantId"`
	CustomerId string `json:"customerId"`
}

type TenantConfigs []TenantConfig
