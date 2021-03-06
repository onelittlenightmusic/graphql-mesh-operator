/*

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
	// json "encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// GraphqlMeshSpec defines the desired state of GraphqlMesh
type GraphqlMeshSpec struct {
	MeshRc          runtime.RawExtension   `json:"meshrc,omitempty"`
	DataSourceNames []string               `json:"dataSourceNames,omitempty"`
	RcConfigMap     GraphqlMeshRcConfigMap `json:"meshrcConfigMap,omitempty"`
	RcSecret        GraphqlMeshRcSecret    `json:"meshrcSecret,omitempty"`
	AsNewDataSource bool                   `json:"asNewDataSource,omitempty"`
}

type GraphqlMeshRcConfigMap struct {
	ConfigMapName string `json:"configMapName"`
}

type GraphqlMeshRcSecret struct {
	SecretName string `json:"secretName"`
}

// GraphqlMeshStatus defines the observed state of GraphqlMesh
type GraphqlMeshStatus struct {
	DeploymentStatus string `json:"deploymentStatus"`
	MeshStatus       string `json:"meshStatus"`
	Endpoint         string `json:"endpoint"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Endpoint",type=string,JSONPath=`.status.endpoint`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.meshStatus`
// +kubebuilder:printcolumn:name="Datasource",type=string,JSONPath=`.spec.meshrc.sources[0].name`

// GraphqlMesh is the Schema for the graphqlmeshes API
type GraphqlMesh struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GraphqlMeshSpec   `json:"spec,omitempty"`
	Status GraphqlMeshStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GraphqlMeshList contains a list of GraphqlMesh
type GraphqlMeshList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GraphqlMesh `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GraphqlMesh{}, &GraphqlMeshList{})
}
