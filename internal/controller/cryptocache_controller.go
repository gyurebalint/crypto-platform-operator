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

package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	databasev1alpha1 "github.com/gyurebalint/crypto-platform-operator/api/v1alpha1"
)

// CryptoCacheReconciler reconciles a CryptoCache object
type CryptoCacheReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=database.crypto.com,resources=cryptocaches,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=database.crypto.com,resources=cryptocaches/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=database.crypto.com,resources=cryptocaches/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CryptoCache object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *CryptoCacheReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	//fetch CryptoCache instance
	cache := &databasev1alpha1.CryptoCache{}
	if err := r.Get(ctx, req.NamespacedName, cache); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//define the deployment of redis
	dep := r.desiredDeployment(cache)

	applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("crypto-operator")}
	if err := r.Patch(ctx, dep, client.Apply, applyOpts...); err != nil {
		l.Error(err, "failed to apply Deployment")
		return ctrl.Result{}, err
	}

	//define the service
	svc := r.desiredService(cache)

	// apply the Service
	if err := r.Patch(ctx, svc, client.Apply, applyOpts...); err != nil {
		l.Error(err, "failed to apply Service")
		return ctrl.Result{}, err
	}

	l.Info("reconciled CryptoCache", "name", cache.Name)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CryptoCacheReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1alpha1.CryptoCache{}).
		Complete(r)
}

func (r *CryptoCacheReconciler) desiredDeployment(cache *databasev1alpha1.CryptoCache) *appsv1.Deployment {
	commonLabels := map[string]string{"app": "crypto-cache", "instance": cache.Name}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cache.Name,
			Namespace: cache.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: commonLabels,
			},
			Replicas: int32Ptr(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: commonLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "redis",
							Image: "redis:7-alpine",
							Ports: []corev1.ContainerPort{
								{ContainerPort: 6379, Name: "redis"},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(cache.Spec.Memory),
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *CryptoCacheReconciler) desiredService(cache *databasev1alpha1.CryptoCache) *corev1.Service {
	commonLabels := map[string]string{"app": "crypto-cache", "instance": cache.Name}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cache.Name,
			Namespace: cache.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: commonLabels,
			Ports: []corev1.ServicePort{
				{
					Port:       6379,
					TargetPort: intstr.FromInt(6379),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}
