/*
Copyright 2021 Wim Henderickx.

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

// FscProxySpec defines the desired state of FscProxy
type FscProxySpec struct {
	// FrontEndPortRange defines the frontEnd Port range of the proxy
	FrontEndPortRange string `json:"frontEndPortRange"`
}

// FscProxyStatus defines the observed state of FscProxy
type FscProxyStatus struct {
	// Targets identifies the targets the proxy handles
	Targets map[string]*Target `json:"targets,omitempty"`
	// FreeFrontEndPortRange identified the free frontEnd port range of the proxy
	FreeFrontEndPortRange *string `json:"freeFrontEndPortRange,omitempty"`
	// LastUpdated identifies when this status was last observed.
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
}

// Target defines the Target attibutes
type Target struct {
	// FrontEndAddress identifies the frontEnd address of the target
	FrontEndAddress string `json:"frontEndAddress,omitempty"`

	// FrontEndCertFile identifies the frontEnd certfile of the target
	FrontEndCertFile string `json:"frontEndCertFile,omitempty"`

	// FrontEndCertKey identifies the frontEnd certkey of the target
	FrontEndCertKey string `json:"frontEndCertKey,omitempty"`

	// BackEndAddress identifies the backEnd address of the target
	BackEndAddress string `json:"backEndAddress,omitempty"`

	// BackEndCertFile identifies the backEnd certfile of the target
	BackEndCertFile string `json:"backEndCertFile,omitempty"`

	// BackEndCertKey identifies the backEnd certkey of the target
	BackEndCertKey string `json:"backEndCertKey,omitempty"`

	// TODO statistics
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// FscProxy is the Schema for the fscproxies API
type FscProxy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FscProxySpec   `json:"spec,omitempty"`
	Status FscProxyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FscProxyList contains a list of FscProxy
type FscProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FscProxy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FscProxy{}, &FscProxyList{})
}
