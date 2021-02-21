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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// NetworkNodeFinalizer is the name of the finalizer added to
	// network node to block delete operations until the physical node can be
	// deprovisioned.
	NetworkNodeFinalizer string = "networknode.fsc.henderiw.be"

	// PausedAnnotation is the annotation that pauses the reconciliation (triggers
	// an immediate requeue)
	PausedAnnotation = "networknode.fsc.henderiw.be/paused"

	// StatusAnnotation is the annotation that keeps a copy of the Status of Network Node
	StatusAnnotation = "networknode.fsc.henderiw.be/status"
)

// DiscoveryStatus defines the states the networkMgmt will report
// the networkNode is having.
type DiscoveryStatus string

const (
	// DiscoveryStatusNone means the state is unknown
	DiscoveryStatusNone DiscoveryStatus = ""

	// DiscoveryStatusNotReady means there is insufficient information available to
	// discover the networkNode
	DiscoveryStatusNotReady DiscoveryStatus = "Not Ready"

	// DiscoveryStatusDiscovery means we are running the discovery on the networkNode to
	// learn about the hardware components
	DiscoveryStatusDiscovery DiscoveryStatus = "Discovery"

	// DiscoveryStatusReady means the networkNode can be consumed
	DiscoveryStatusReady DiscoveryStatus = "Ready"

	// DiscoveryStatusDeleting means we are in the process of cleaning up the networkNode
	// ready for deletion
	DiscoveryStatusDeleting DiscoveryStatus = "Deleting"
)

// OperationalStatus represents the state of the network node
type OperationalStatus string

const (
	// OperationalStatusNone means the state is unknown
	OperationalStatusNone OperationalStatus = ""

	// OperationalStatusUp is the status value for when the network node is
	// configured correctly and is manageable/operational.
	OperationalStatusUp OperationalStatus = "Up"

	// OperationalStatusDown is the status value for when the network node
	// has any sort of error.
	OperationalStatusDown OperationalStatus = "Down"
)

// ErrorType indicates the class of problem that has caused the Network Node resource
// to enter an error state.
type ErrorType string

const (
	// NoneError is an error type when no error exists
	NoneError ErrorType = ""
	// TargetError is an error condition occurring when the
	// target supplied are not correct or not existing
	TargetError ErrorType = "target error"
	// CredentialError is an error condition occurring when the
	// credentials supplied are not correct or not existing
	CredentialError ErrorType = "credential error"
	// DiscoveryError is an error condition occurring when the
	// controller is unable to connect to the Network Node
	DiscoveryError ErrorType = "discovery error"
	// NetwMgmtFactoryError is an error condition occuring when the controller
	// fails to register with the netwkmgmt factory. Protocol spec issue
	NetwMgmtFactoryError ErrorType = "netwmgmt factor error"
	// ProvisioningError is an error condition occuring when the controller
	// fails to provision or deprovision the networkNode.
	ProvisioningError ErrorType = "provisioning error"
)

// NetworkNodeSpec defines the desired state of NetworkNode
type NetworkNodeSpec struct {
	// Target defines how we connect to the network node
	Target TargetDetails `json:"target,omitempty"`
	// ConsumerRef can be used to store information about something
	// that is using a network Node. When it is not empty, the Network Node is
	// considered "in use".
	ConsumerRef *corev1.ObjectReference `json:"consumerRef,omitempty"`
}

// TargetDetails contains the information necessary to communicate with
// the network node.
type TargetDetails struct {
	// Protocol used to communicate to the target network node
	Protocol string `json:"protocol,omitempty"`

	// Proxy used to communicate to the target network node
	Proxy string `json:"proxy,omitempty"`

	// Address holds the IP:port for accessing the network node
	Address string `json:"address"`

	// The name of the secret containing the credentials (requires
	// keys "username" and "password").
	CredentialsName string `json:"credentialsName"`

	// The name of the secret containing the credentials (requires
	// keys "TLSCA" and "TLSCert", " TLSKey").
	TLSCredentialsName string `json:"tlsCredentialsName,omitempty"`

	// SkipVerify disables verification of server certificates when using
	// HTTPS to connect to the Target. This is required when the server
	// certificate is self-signed, but is insecure because it allows a
	// man-in-the-middle to intercept the connection.
	SkipVerify bool `json:"skpVerify,omitempty"`

	// Insecure runs the grpc call in an insecure manner
	Insecure bool `json:"insecure,omitempty"`

	// Encoding
	Encoding string `json:"encoding,omitempty"`
}

// OperationMetric contains metadata about an operation (inspection,
// provisioning, etc.) used for tracking metrics.
type OperationMetric struct {
	// +nullable
	Start metav1.Time `json:"start,omitempty"`
	// +nullable
	End metav1.Time `json:"end,omitempty"`
}

// Duration returns the length of time that was spent on the
// operation. If the operation is not finished, it returns 0.
func (om OperationMetric) Duration() time.Duration {
	if om.Start.IsZero() {
		return 0
	}
	return om.End.Time.Sub(om.Start.Time)
}

// OperationHistory holds information about operations performed on a
// networkNode.
type OperationHistory struct {
	Discover OperationMetric `json:"discover,omitempty"`
	Set      OperationMetric `json:"set,omitempty"`
	Get      OperationMetric `json:"get,omitempty"`
}

// NetworkNodeStatus defines the observed state of NetworkNode
type NetworkNodeStatus struct {
	// DiscoveryStatus holds the discovery status of the networkNode
	// +kubebuilder:validation:Enum="";Enabled;Disabled;Discovery;Deleting
	DiscoveryStatus DiscoveryStatus `json:"discoveryStatus"`

	// OperationalStatus holds the operational status of the networkNode
	// +kubebuilder:validation:Enum="";Enabled;Disabled
	OperationalStatus OperationalStatus `json:"operationalStatus"`

	// ErrorType indicates the type of failure encountered when the
	// OperationalStatus is OperationalStatusDown
	// +kubebuilder:validation:Enum="";target error;credential error;discovery error;netwmgmt factor error;provisioning error
	ErrorType ErrorType `json:"errorType,omitempty"`

	// LastUpdated identifies when this status was last observed.
	// +optional
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`

	// The HardwareDetails discovered on the Network Node.
	HardwareDetails HardwareDetails `json:"hardwareDetails,omitempty"`

	// the last credentials we were able to validate as working
	GoodCredentials CredentialsStatus `json:"goodCredentials,omitempty"`

	// the last credentials we sent to the provisioning backend
	TriedCredentials CredentialsStatus `json:"triedCredentials,omitempty"`

	// the last error message reported by the provisioning subsystem
	ErrorMessage string `json:"errorMessage"`

	// OperationHistory holds information about operations performed
	// on this host.
	OperationHistory OperationHistory `json:"operationHistory"`

	// ErrorCount records how many times the host has encoutered an error since the last successful operation
	// +kubebuilder:default:=0
	ErrorCount int `json:"errorCount"`
}

// HardwareDetails collects all of the information about hardware
// discovered on the Network Node.
type HardwareDetails struct {
	// the Kind of hardware
	Kind string `json:"kind,omitempty"`
	// the Mac address of the hardware
	MacAddress string `json:"macAddress,omitempty"`
	// the Serial Number of the hardware
	SerialNumber string `json:"serialNumber,omitempty"`
}

// CredentialsStatus contains the reference and version of the last
// set of credentials the controller was able to validate.
type CredentialsStatus struct {
	Reference *corev1.SecretReference `json:"credentials,omitempty"`
	Version   string                  `json:"credentialsVersion,omitempty"`
}

// Match compares the saved status information with the name and
// content of a secret object.
func (cs CredentialsStatus) Match(secret corev1.Secret) bool {
	switch {
	case cs.Reference == nil:
		return false
	case cs.Reference.Name != secret.ObjectMeta.Name:
		return false
	case cs.Reference.Namespace != secret.ObjectMeta.Namespace:
		return false
	case cs.Version != secret.ObjectMeta.ResourceVersion:
		return false
	}
	return true
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// NetworkNode is the Schema for the networknodes API
type NetworkNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkNodeSpec   `json:"spec,omitempty"`
	Status NetworkNodeStatus `json:"status,omitempty"`
}

// HasTargetDetails returns true if the Target details are set
func (nn *NetworkNode) HasTargetDetails() bool {
	return nn.Spec.Target.Address != "" || nn.Spec.Target.CredentialsName != ""
}

// SetOperationalStatus updates the OperationalStatus field and returns
// true when a change is made or false when no change is made.
func (nn *NetworkNode) SetOperationalStatus(status OperationalStatus) bool {
	if nn.Status.OperationalStatus != status {
		nn.Status.OperationalStatus = status
		return true
	}
	return false
}

// SetDiscoveryStatus updates the Discovery Status field and returns
// true when a change is made or false when no change is made.
func (nn *NetworkNode) SetDiscoveryStatus(status DiscoveryStatus) bool {
	if nn.Status.DiscoveryStatus != status {
		nn.Status.DiscoveryStatus = status
		return true
	}
	return false
}

// GetOperationalStatus return the operational status
func (nn *NetworkNode) GetOperationalStatus() string {
	return string(nn.Status.OperationalStatus)
}

// GetDiscoveryStatus return the discovery status
func (nn *NetworkNode) GetDiscoveryStatus() string {
	return string(nn.Status.DiscoveryStatus)
}

// SetErrorType updates the Error Type field and returns
// true when a change is made or false when no change is made.
func (nn *NetworkNode) SetErrorType(errorType ErrorType) bool {
	if nn.Status.ErrorType != errorType {
		nn.Status.ErrorType = errorType
		return true
	}
	return false
}

// SetHardwareDetails updates the hardware details
func (nn *NetworkNode) SetHardwareDetails(hwDetails *HardwareDetails) {
	if nn.Status.HardwareDetails.Kind != hwDetails.Kind {
		nn.Status.HardwareDetails.Kind = hwDetails.Kind
	}

	if nn.Status.HardwareDetails.SerialNumber != hwDetails.SerialNumber {
		nn.Status.HardwareDetails.SerialNumber = hwDetails.SerialNumber
	}

	if nn.Status.HardwareDetails.MacAddress != hwDetails.MacAddress {
		nn.Status.HardwareDetails.MacAddress = hwDetails.MacAddress
	}

}

// +kubebuilder:object:root=true

// NetworkNodeList contains a list of NetworkNode
type NetworkNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkNode `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkNode{}, &NetworkNodeList{})
}

// CredentialsKey returns a NamespacedName suitable for loading the
// Secret containing the credentials associated with the host.
func (nn *NetworkNode) CredentialsKey() types.NamespacedName {
	return types.NamespacedName{
		Name:      nn.Spec.Target.CredentialsName,
		Namespace: nn.ObjectMeta.Namespace,
	}
}

// NewEvent creates a new event associated with the object and ready
// to be published to the kubernetes API.
func (nn *NetworkNode) NewEvent(reason, message string) corev1.Event {
	t := metav1.Now()
	return corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: reason + "-",
			Namespace:    nn.ObjectMeta.Namespace,
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:       "NetworkNode",
			Namespace:  nn.Namespace,
			Name:       nn.Name,
			UID:        nn.UID,
			APIVersion: GroupVersion.String(),
		},
		Reason:  reason,
		Message: message,
		Source: corev1.EventSource{
			Component: "fsc-discovery-controller",
		},
		FirstTimestamp:      t,
		LastTimestamp:       t,
		Count:               1,
		Type:                corev1.EventTypeNormal,
		ReportingController: "fsc.henderiw.be/fsc-discovery-controller",
		Related:             nn.Spec.ConsumerRef,
	}
}

// OperationMetricForState returns a pointer to the metric for the given
// discovery state.
func (nn *NetworkNode) OperationMetricForState(operation DiscoveryStatus) (metric *OperationMetric) {
	history := &nn.Status.OperationHistory
	switch operation {
	case DiscoveryStatusDiscovery:
		metric = &history.Discover
	}
	return
}
