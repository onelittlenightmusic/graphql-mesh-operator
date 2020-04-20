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
	json "encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// type MeshRc struct {
// 	Data map[string]interface{} `json:"test,"`
// }

// GraphqlMeshSpec defines the desired state of GraphqlMesh
type GraphqlMeshSpec struct {
	Rc          json.RawMessage        `json:"meshrc,omitempty"`
	RcConfigMap GraphqlMeshRcConfigMap `json:"meshrcConfigMap,omitempty"`
	RcSecret    GraphqlMeshRcSecret    `json:"meshrcSecret,omitempty"`
}

type GraphqlMeshRcConfigMap struct {
	ConfigMapName string `json:"configMapName"`
}

type GraphqlMeshRcSecret struct {
	SecretName string `json:"secretName"`
}

// // MarshalJSON marshals the MeshRc data to a JSON blob.
// func (v MeshRc) MarshalJSON() ([]byte, error) {
// 	return json.Marshal(v.Data)
// }

// // UnmarshalJSON sets the MeshRc to a copy of data.
// func (v *MeshRc) UnmarshalJSON(data []byte) error {
// 	var out map[string]interface{}
// 	err := json.Unmarshal(data, &out)
// 	if err != nil {
// 		return err
// 	}
// 	v.Data = out
// 	return nil
// }

// GraphqlMeshStatus defines the observed state of GraphqlMesh
type GraphqlMeshStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

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
