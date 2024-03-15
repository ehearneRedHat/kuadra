/*
Copyright 2023.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AwsAccountSpec defines the desired state of AwsAccount
type AwsAccountSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	UserName string   `json:"userName"`
	Groups   []string `json:"groups"`
}

// AwsAccountStatus defines the observed state of AwsAccount
type AwsAccountStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	UserCreated bool `json:"userCreated"`

	// +optional
	LoginProfileCreated bool `json:"loginProfileCreated"`

	// +optional
	AccessKeyCreated bool `json:"accessKeyCreated"`

	// +optional
	UserGroups []string `json:"userGroups"`

	// +optional
	NamespaceCreated bool `json:"namespaceCreated"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// AwsAccount is the Schema for the awsaccounts API
type AwsAccount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AwsAccountSpec   `json:"spec,omitempty"`
	Status AwsAccountStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AwsAccountList contains a list of AwsAccount
type AwsAccountList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AwsAccount `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AwsAccount{}, &AwsAccountList{})
}
