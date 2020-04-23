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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	ptr "k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	controller = ".metadata.controller"
	apiGVStr   = meshv1alpha1.GroupVersion.String()
)

const (
	defaultServeHostnameServicePort = 4000
)

// GraphqlMeshReconciler reconciles a GraphqlMesh object
type GraphqlMeshReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=mesh.graphql-mesh-operator.io,resources=graphqlmeshes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mesh.graphql-mesh-operator.io,resources=graphqlmeshes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps/status,verbs=get;update;patch
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

	// if err := r.cleanupOwnedResources(ctx, log, &graphqlMesh); err != nil {
	// 	log.Error(err, "unable to clean up existing GraphqlMesh resources")
	// 	return ctrl.Result{}, nil
	// }
	configMapName := fmt.Sprintf("%s-meshrc", graphqlMesh.Name)
	var configMap corev1.ConfigMap
	if err := r.Get(ctx, client.ObjectKey{Namespace: graphqlMesh.Namespace, Name: configMapName}, &configMap); apierrors.IsNotFound(err) {
		configMap, err := r.constructMeshrc(&graphqlMesh, configMapName)
		if err != nil {
			log.Error(err, "unable to construct ConfigMap for meshrc")
			return ctrl.Result{}, nil
		}

		if err := r.Create(ctx, configMap); err != nil {
			log.Error(err, "unable to create configMap for GraphqlMesh", "configMap", configMap)
			return ctrl.Result{}, err
		}
	}

	name := fmt.Sprintf("%s", graphqlMesh.Name)
	var deploy appsv1.Deployment
	if err := r.Get(ctx, client.ObjectKey{Namespace: graphqlMesh.Namespace, Name: name}, &deploy); apierrors.IsNotFound(err) {
		deploy, err := r.constructDeployment(&graphqlMesh, name)
		if err != nil {
			log.Error(err, "unable to construct Deployment from meshrc")
			return ctrl.Result{}, nil
		}
		if err := r.Create(ctx, deploy); err != nil {
			log.Error(err, "unable to create Deployment for GraphqlMesh", "deployment", deploy)
			return ctrl.Result{}, err
		}
	}

	serviceName := fmt.Sprintf("%s-svc", graphqlMesh.Name)
	var service corev1.Service
	if err := r.Get(ctx, client.ObjectKey{Namespace: graphqlMesh.Namespace, Name: serviceName}, &service); apierrors.IsNotFound(err) {
		service, err := r.constructService(&graphqlMesh, serviceName, name)
		if err != nil {
			log.Error(err, "unable to construct Service from meshrc")
			return ctrl.Result{}, nil
		}
		if err := r.Create(ctx, service); err != nil {
			log.Error(err, "unable to create Service for GraphqlMesh", "service", service)
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

func (r *GraphqlMeshReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&appsv1.Deployment{}, controller, func(rawObj runtime.Object) []string {
		deploy := rawObj.(*appsv1.Deployment)
		owner := metav1.GetControllerOf(deploy)
		if owner == nil {
			return nil
		}
		if owner.APIVersion != apiGVStr || owner.Kind != "GraphqlMesh" {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		return err
	}
	if err := mgr.GetFieldIndexer().IndexField(&corev1.ConfigMap{}, controller, func(rawObj runtime.Object) []string {
		cm := rawObj.(*corev1.ConfigMap)
		owner := metav1.GetControllerOf(cm)
		if owner == nil {
			return nil
		}
		if owner.APIVersion != apiGVStr || owner.Kind != "GraphqlMesh" {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&meshv1alpha1.GraphqlMesh{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

func (r *GraphqlMeshReconciler) cleanupOwnedResources(ctx context.Context, log logr.Logger, graphqlMesh *meshv1alpha1.GraphqlMesh) error {
	var deployList appsv1.DeploymentList
	if err := r.List(ctx, &deployList, client.InNamespace(graphqlMesh.Namespace), client.MatchingFields{controller: graphqlMesh.Name}); err != nil {
		log.Error(err, "unable to list of deployment")
		return err
	}
	deleted := 0
	// name := fmt.Sprintf("%s-deploy", graphqlMesh.Name)
	// configMapName := fmt.Sprintf("%s-meshrc", graphqlMesh.Name)
	for _, depl := range deployList.Items {
		// if depl.Name != name {
		// 	continue
		// }

		if err := r.Client.Delete(ctx, &depl); err != nil {
			log.Error(err, "failed to delete Deployment resource")
			return err
		}

		r.Recorder.Eventf(graphqlMesh, corev1.EventTypeNormal, "Deleted", "Deleted deployment %q", depl.Name)
		deleted++
	}
	log.V(1).Info("finished cleaning up old Deployment resources", "number_deleted", deleted)
	deletedCM := 0
	var configMapList corev1.ConfigMapList
	if err := r.List(ctx, &configMapList, client.InNamespace(graphqlMesh.Namespace), client.MatchingFields{controller: graphqlMesh.Name}); err != nil {
		log.Error(err, "unable to list of configmap")
		return err
	}
	for _, configMap := range configMapList.Items {
		// if configMap.Name != configMapName {
		// 	continue
		// }

		if err := r.Client.Delete(ctx, &configMap); err != nil {
			log.Error(err, "failed to delete ConfigMap resource")
			return err
		}

		r.Recorder.Eventf(graphqlMesh, corev1.EventTypeNormal, "Deleted", "Deleted configmap %q", configMap.Name)
		deletedCM++
	}
	log.V(1).Info("finished cleaning up old ConfigMap resources", "number_deleted", deletedCM)
	return nil
}

func (r *GraphqlMeshReconciler) constructDeployment(graphqlMesh *meshv1alpha1.GraphqlMesh, name string) (*appsv1.Deployment, error) {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: graphqlMesh.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": name,
					"app":  "graphql-mesh",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"name": name,
						"app":  "graphql-mesh",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "web",
							Image: "hiroyukiosaki/graphql-mesh:v0.1.10-all",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 4000,
								},
							},
						},
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(graphqlMesh, deploy, r.Scheme); err != nil {
		return nil, err
	}

	return deploy, nil
}

func (r *GraphqlMeshReconciler) constructService(graphqlMesh *meshv1alpha1.GraphqlMesh, name string, deployName string) (*corev1.Service, error) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: graphqlMesh.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Port:       int32(defaultServeHostnameServicePort),
				TargetPort: intstr.FromInt(4000),
				Protocol:   corev1.ProtocolTCP,
			}},
			Selector: map[string]string{
				"name": deployName,
				"app":  "graphql-mesh",
			},
		},
	}

	if err := ctrl.SetControllerReference(graphqlMesh, service, r.Scheme); err != nil {
		return nil, err
	}

	return service, nil
}

func (r *GraphqlMeshReconciler) constructConfigMap(graphqlMesh *meshv1alpha1.GraphqlMesh, data *map[string]string, name string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: graphqlMesh.Namespace,
		},
		Data: *data,
	}

	if err := ctrl.SetControllerReference(graphqlMesh, configMap, r.Scheme); err != nil {
		return nil, err
	}
	return configMap, nil
}

func (r *GraphqlMeshReconciler) constructMeshrc(graphqlMesh *meshv1alpha1.GraphqlMesh, configMapName string) (*corev1.ConfigMap, error) {
	data := &map[string]string{
		".meshrc": r.Stringify(&graphqlMesh.Spec.Rc),
	}
	return r.constructConfigMap(graphqlMesh, data, configMapName)
}

func (r *GraphqlMeshReconciler) Stringify(a *runtime.RawExtension) string {
	byt, _ := json.Marshal(a)
	return string(byt)
}
