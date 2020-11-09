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

func (r *GraphqlMeshReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("graphqlmesh", req.NamespacedName)

	// Get graphqlMesh object
	var graphqlMesh meshv1alpha1.GraphqlMesh
	if err := r.Get(ctx, req.NamespacedName, &graphqlMesh); err != nil {
		log.Error(err, "unable to fetch GraphqlMesh")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	configMapName := fmt.Sprintf("%s-meshrc", graphqlMesh.Name)
	var configMap corev1.ConfigMap
	if err := r.Get(ctx, client.ObjectKey{Namespace: graphqlMesh.Namespace, Name: configMapName}, &configMap); apierrors.IsNotFound(err) {
		if err := r.constructMeshrc(&graphqlMesh, configMapName, &configMap); err != nil {
			log.Error(err, "unable to construct ConfigMap for meshrc")
			return ctrl.Result{}, err
		}

		if err := r.Create(ctx, &configMap); err != nil {
			log.Error(err, "unable to create configMap for GraphqlMesh", "configMap", configMap)
			return ctrl.Result{}, err
		}
	}

	// configMapSrcName := fmt.Sprintf("%s-src", graphqlMesh.Name)
	// var configMapSrc corev1.ConfigMap
	// if err := r.Get(ctx, client.ObjectKey{Namespace: graphqlMesh.Namespace, Name: configMapSrcName}, &configMapSrc); apierrors.IsNotFound(err) {
	// 	configMapSrc, err := r.constructSrc(&graphqlMesh, configMapSrcName)
	// 	if err != nil {
	// 		log.Error(err, "unable to construct ConfigMap for src")
	// 		return ctrl.Result{}, nil
	// 	}

	// 	if err := r.Create(ctx, configMapSrc); err != nil {
	// 		log.Error(err, "unable to create configMap for GraphqlMesh", "configMap", configMapSrc)
	// 		return ctrl.Result{}, err
	// 	}
	// }

	name := fmt.Sprintf("%s", graphqlMesh.Name)
	var deploy appsv1.Deployment
	if err := r.Get(ctx, client.ObjectKey{Namespace: graphqlMesh.Namespace, Name: name}, &deploy); apierrors.IsNotFound(err) {
		if err := r.constructDeployment(&graphqlMesh, name, configMapName, &deploy); err != nil {
			log.Error(err, "unable to construct Deployment")
			return ctrl.Result{}, nil
		}
		if err := r.Create(ctx, &deploy); err != nil {
			log.Error(err, "unable to create Deployment for GraphqlMesh", "deployment", deploy)
			return ctrl.Result{}, err
		}
		graphqlMesh.Status.DeploymentStatus = "Ok"
		r.Update(ctx, &graphqlMesh)
	}

	serviceName := fmt.Sprintf("%s-svc", graphqlMesh.Name)
	var service corev1.Service
	if err := r.Get(ctx, client.ObjectKey{Namespace: graphqlMesh.Namespace, Name: serviceName}, &service); apierrors.IsNotFound(err) {
		if err := r.constructService(&graphqlMesh, serviceName, name, &service); err != nil {
			log.Error(err, "unable to construct Service")
			return ctrl.Result{}, nil
		}
		if err := r.Create(ctx, &service); err != nil {
			log.Error(err, "unable to create Service for GraphqlMesh", "service", service)
			return ctrl.Result{}, err
		}

		// set .status.meshStatus to Ok
		graphqlMesh.Status.MeshStatus = "Ok"
		graphqlMesh.Status.Endpoint = fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, graphqlMesh.Namespace)
		r.Update(ctx, &graphqlMesh)
	}
	return ctrl.Result{}, nil
}

func (r *GraphqlMeshReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := MakeIndex(mgr, &appsv1.Deployment{}); err != nil {
		return err
	}
	if err := MakeIndex(mgr, &corev1.ConfigMap{}); err != nil {
		return err
	}
	if err := MakeIndex(mgr, &corev1.Service{}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&meshv1alpha1.GraphqlMesh{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func MakeIndex(mgr ctrl.Manager, typeInstance runtime.Object) error {
	return mgr.GetFieldIndexer().IndexField(typeInstance, controller, GetOwnerName)
}

func CheckOwner(obj metav1.Object) (bool, *metav1.OwnerReference) {
	owner := metav1.GetControllerOf(obj)
	noowner := owner == nil || owner.APIVersion != apiGVStr || owner.Kind != "GraphqlMesh"
	return !noowner, owner
}

func GetOwnerName(rawObj runtime.Object) []string {
	if owned, owner := CheckOwner(rawObj.(metav1.Object)); owned {
		return []string{owner.Name}
	} else {
		return nil
	}
}

func (r *GraphqlMeshReconciler) constructDeployment(graphqlMesh *meshv1alpha1.GraphqlMesh, name string, configMapName string, deploy *appsv1.Deployment) error {
	deployRtn := appsv1.Deployment{
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
							Image: "hiroyukiosaki/graphql-mesh:latest-all",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 4000,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "meshrc-cm",
									MountPath: "/work/.meshrc.yaml",
									SubPath:   ".meshrc.yaml",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "meshrc-cm",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configMapName,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(graphqlMesh, &deployRtn, r.Scheme); err != nil {
		return err
	}

	*deploy = deployRtn
	return nil
}

func (r *GraphqlMeshReconciler) constructService(graphqlMesh *meshv1alpha1.GraphqlMesh, name string, deployName string, service *corev1.Service) error {
	serviceRtn := corev1.Service{
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

	if err := ctrl.SetControllerReference(graphqlMesh, &serviceRtn, r.Scheme); err != nil {
		return err
	}

	*service = serviceRtn
	return nil
}

func (r *GraphqlMeshReconciler) constructConfigMap(graphqlMesh *meshv1alpha1.GraphqlMesh, data *map[string]string, name string, cm *corev1.ConfigMap) error {
	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: graphqlMesh.Namespace,
		},
		Data: *data,
	}

	if err := ctrl.SetControllerReference(graphqlMesh, &configMap, r.Scheme); err != nil {
		return err
	}
	*cm = configMap
	return nil
}

func (r *GraphqlMeshReconciler) constructMeshrc(graphqlMesh *meshv1alpha1.GraphqlMesh, configMapName string, cm *corev1.ConfigMap) error {
	ctx := context.Background()
	log := r.Log.WithValues("graphqlmesh", graphqlMesh.Namespace)
	var returnMesh interface{} = graphqlMesh.Spec.MeshRc

	if dataSourceNames := graphqlMesh.Spec.DataSourceNames; len(dataSourceNames) > 0 {
		var sources []map[string]interface{}

		for _, dataSourceName := range dataSourceNames {
			var dataSource meshv1alpha1.DataSource
			if err := r.Get(ctx, client.ObjectKey{Namespace: graphqlMesh.Namespace, Name: dataSourceName}, &dataSource); err != nil {
				return err
			}
			sources = append(sources,
				map[string]interface{}{
					"name": dataSourceName,
					"handler": map[string]interface{}{
						dataSource.Spec.Type: &dataSource.Spec.HandlerConfig,
					},
				},
			)
		}

		returnMesh = map[string]interface{}{
			"sources": sources,
		}

		originalMesh := graphqlMesh.Spec.MeshRc.Raw
		log.Info(string(graphqlMesh.Spec.MeshRc.Raw))

		if string(originalMesh) != "" {
			var unmarshaled interface{}
			if err := json.Unmarshal(originalMesh, &unmarshaled); err != nil {
				return err
			}
			returnMesh = merge(unmarshaled, returnMesh)
		}
	}

	data := &map[string]string{
		".meshrc.yaml": r.Stringify(returnMesh),
	}
	return r.constructConfigMap(graphqlMesh, data, configMapName, cm)
}

func (r *GraphqlMeshReconciler) Stringify(a interface{}) string {
	byt, _ := json.Marshal(&a)
	return string(byt)
}

func merge(x1, x2 interface{}) interface{} {
	switch x1 := x1.(type) {
	case map[string]interface{}:
		x2, ok := x2.(map[string]interface{})
		if !ok {
			return x1
		}
		for k, v2 := range x2 {
			if v1, ok := x1[k]; ok {
				x1[k] = merge(v1, v2)
			} else {
				x1[k] = v2
			}
		}
	case nil:
		if x2, ok := x2.(map[string]interface{}); ok {
			x1 = x2
		}
	}
	return x1
}
