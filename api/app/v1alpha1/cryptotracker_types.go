/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CryptoTrackerSpec defines the desired state of CryptoTracker
type CryptoTrackerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	Image string `json:"image"`

	// +kubebuilder:validation:Required
	Symbol string `json:"symbol"`

	// DB is the name of the CryptoDatabase to connect to (Optional)
	// +optional
	DB string `json:"db,omitempty"`

	// Cache is the name of the CryptoCache to connect to (Optional)
	// +optional
	Cache string `json:"cache,omitempty"`
}

// CryptoTrackerStatus defines the observed state of CryptoTracker
type CryptoTrackerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// CryptoTracker is the Schema for the cryptotrackers API
type CryptoTracker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CryptoTrackerSpec   `json:"spec,omitempty"`
	Status CryptoTrackerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CryptoTrackerList contains a list of CryptoTracker
type CryptoTrackerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CryptoTracker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CryptoTracker{}, &CryptoTrackerList{})
}
