/*
Copyright 2022.

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

package v1beta1

import (
	"fmt"
	"github.com/git-czy/cluster-api-metalnode/pkg/remote"
	"github.com/git-czy/cluster-api-metalnode/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net"
	"strings"
)

type InitializationState string

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MetalNodeSpec defines the desired state of MetalNode
type MetalNodeSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// NodeName is the name of metal node
	// +optional
	NodeName string `json:"nodeName,omitempty"`

	// NodeEndPoint is the endpoint of MetalNode
	NodeEndPoint Endpoint `json:"nodeEndPoint"`

	// InitializedCmd
	// +optional
	InitializationCmd remote.Commands `json:"initializationCmd,omitempty"`
}

type Endpoint struct {
	// ssh Host
	Host string `json:"host"`

	// SSHAuth denotes ssh auth
	SSHAuth Auth `json:"sshAuth"`
}

type Auth struct {
	// User denotes ssh connect user
	User string `json:"user"`

	// Password denotes ssh connect password
	// +optional
	Password string `json:"password,omitempty"`

	// SSHKey denotes ssh connect sshKey
	// +optional
	SSHKey string `json:"sshKey,omitempty"`

	// ssh Port.
	Port int `json:"port"`
}

// MetalNodeStatus defines the observed state of MetalNode
type MetalNodeStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Initialized denotes if this node is init k8s env(kubeadm,kubectl,kubelet,iptables ...)
	// +optional
	InitializationState InitializationState `json:"InitializationState"`

	// InitializedFailureReason denotes run shell command standard stderr
	// This FailureReason always occurs,but not necessarily the reason for the real initialization failure
	// maybe you can get some information when an error occurs in the deployment
	// +optional
	InitializationFailureReason []string `json:"InitializationFailureReason,omitempty"`

	// CheckFailureReason denotes if this node is checked in(docker kubectl kubelet kubeadm)
	// +optional
	CheckFailureReason []string `json:"CheckFailureReason,omitempty"`

	// Role denotes the role of this node ,such as master,worker,etcd,load-balance...
	// +optional
	Role []string `json:"role,omitempty"`

	// RefCluster denotes the name of the cluster which this node belongs to
	// +optional
	RefCluster string `json:"refCluster,omitempty"`

	// DataSecretName denotes the name of the secret which stores the data of this bootstrap data
	// +optional
	DataSecretName string `json:"dataSecretName,omitempty"`

	// Bootstrapped denotes if this node is bootstrapped
	Bootstrapped bool `json:"bootstrapped"`

	// BootstrappedFailureReason denotes run bootstrap shell command standard stderr
	BootstrapFailureReason []string `json:"bootstrapFailureReason,omitempty"`

	// Ready denotes this metal node is ready to init | join a k8s cluster
	Ready bool `json:"ready"`
}

func (e Endpoint) Validate() error {
	if e.Host != "" {
		if host := net.ParseIP(e.Host); host == nil {
			return fmt.Errorf("Endpoint's host %s not a validate IP,neither IPV4 nor IPV6 ", e.Host)
		}
	}
	return nil
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:resource:shortName=mn
// +kubebuilder:printcolumn:name="READY",type="boolean",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.InitializationState"
// +kubebuilder:printcolumn:name="ROLE",type="string",JSONPath=".status.Role"
// +kubebuilder:printcolumn:name="CLUSTER",type="string",JSONPath=".status.RefCluster"

// MetalNode is the Schema for the metalnodes API
type MetalNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MetalNodeSpec   `json:"spec,omitempty"`
	Status MetalNodeStatus `json:"status,omitempty"`
}

// SetRole set MetalNode status role
func (mn *MetalNode) SetRole(role string) {
	mn.Status.Role = append(mn.Status.Role, role)
}

func (mn *MetalNode) HasRole(role string) bool {
	return utils.SliceContainsString(mn.Status.Role, role)
}

// GetRefCluster get MetalNode status refCluster
func (mn *MetalNode) GetRefCluster() string {
	return mn.Status.RefCluster
}

// ContainRole check if the role is in the role list
func (mn *MetalNode) ContainRole(role string) bool {
	return mn.Status.Role != nil && len(mn.Status.Role) > 0 && strings.Contains(strings.Join(mn.Status.Role, ","), role)
}

// IsReady check if the metal node is ready
func (mn *MetalNode) IsReady() bool {
	return mn.Status.Ready
}

// ResetMetalNode reset MetalNode status after ref cluster delete node
func (mn *MetalNode) ResetMetalNode() {
	mn.Status.Role = nil
	mn.Status.RefCluster = ""
	mn.Status.DataSecretName = ""
	mn.Status.Bootstrapped = false
	mn.Status.DataSecretName = ""
}

//+kubebuilder:object:root=true

// MetalNodeList contains a list of MetalNode
type MetalNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MetalNode `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MetalNode{}, &MetalNodeList{})
}
