package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CryptoDatabaseSpec defines the desired state of CryptoDatabase
type CryptoDatabaseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// StorageSize defines the size of the PVC (e.g., "10Gi")
	// +kubebuilder:validation:Required
	StorageSize string `json:"storageSize"`
}

// CryptoDatabaseStatus defines the observed state of CryptoDatabase
type CryptoDatabaseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CryptoDatabase is the Schema for the cryptodatabases API
type CryptoDatabase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CryptoDatabaseSpec   `json:"spec,omitempty"`
	Status CryptoDatabaseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CryptoDatabaseList contains a list of CryptoDatabase
type CryptoDatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CryptoDatabase `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CryptoDatabase{}, &CryptoDatabaseList{})
}
