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

package controllers

import (
	"context"
	json "encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	meshv1alpha1 "graphql-mesh-operator.io/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ptr "k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	controller = "metadata.ownerReferences.name"
)

// GraphqlMeshReconciler reconciles a GraphqlMesh object
type GraphqlMeshReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=mesh.graphql-mesh-operator.io,resources=graphqlmeshes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mesh.graphql-mesh-operator.io,resources=graphqlmeshes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
func (r *GraphqlMeshReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("graphqlmesh", req.NamespacedName)

	// your logic here
	var graphqlMesh meshv1alpha1.GraphqlMesh
	if err := r.Get(ctx, req.NamespacedName, &graphqlMesh); err != nil {
		log.Error(err, "unable to fetch GraphqlMesh")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.V(1).Info("test", "graphqlMesh", graphqlMesh)

	// var deployList appsv1.DeploymentList
	// if err := r.List(ctx, &deployList, client.InNamespace(req.Namespace), client.MatchingFields{controller: req.Name}); err != nil {
	// 	log.Error(err, "unable to list child Jobs")
	// 	return ctrl.Result{}, err
	// }
	// log.V(1).Info("test", "deployments", deployList)

	deploy, configMap, err := r.constructDeployment(&graphqlMesh)
	if err != nil {
		log.Error(err, "unable to construct Deployment from meshrc")
		return ctrl.Result{}, nil
	}

	if err := r.Create(ctx, configMap); err != nil {
		log.Error(err, "unable to create configMap for GraphqlMesh", "configMap", configMap)
		return ctrl.Result{}, err
	}

	if err := r.Create(ctx, deploy); err != nil {
		log.Error(err, "unable to create Deployment for GraphqlMesh", "deployment", deploy)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *GraphqlMeshReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&meshv1alpha1.GraphqlMesh{}).
		Complete(r)
}

func (r *GraphqlMeshReconciler) constructDeployment(graphqlMesh *meshv1alpha1.GraphqlMesh) (*appsv1.Deployment, *corev1.ConfigMap, error) {
	name := fmt.Sprintf("%s-deploy", graphqlMesh.Name)

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: graphqlMesh.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.Int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "web",
							Image: "nginx:1.12",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	configMapName := fmt.Sprintf("%s-meshrc", graphqlMesh.Name)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: graphqlMesh.Namespace,
		},
		Data: map[string]string{
			".meshrc": r.Stringify(&graphqlMesh.Spec.Rc),
		},
	}

	if err := ctrl.SetControllerReference(graphqlMesh, deploy, r.Scheme); err != nil {
		return nil, nil, err
	}

	if err := ctrl.SetControllerReference(graphqlMesh, configMap, r.Scheme); err != nil {
		return nil, nil, err
	}

	return deploy, configMap, nil
}

func (r *GraphqlMeshReconciler) Stringify(a *json.RawMessage) string {
	byt, _ := json.Marshal(a)
	return string(byt)
}
