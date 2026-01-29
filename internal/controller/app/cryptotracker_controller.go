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

package app

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appv1alpha1 "github.com/gyurebalint/crypto-platform-operator/api/app/v1alpha1"

	// Import the DB/Cache APIs (Group: database)
	// We alias this to 'databasev1alpha1' to avoid confusion
	databasev1alpha1 "github.com/gyurebalint/crypto-platform-operator/api/v1alpha1"
)

// CryptoTrackerReconciler reconciles a CryptoTracker object
type CryptoTrackerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=app.crypto.com,resources=cryptotrackers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.crypto.com,resources=cryptotrackers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=app.crypto.com,resources=cryptotrackers/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=database.crypto.com,resources=cryptodatabases,verbs=get;list;watch
// +kubebuilder:rbac:groups=database.crypto.com,resources=cryptocaches,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CryptoTracker object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *CryptoTrackerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	//fetch the crypto tracker instance
	tracker := &appv1alpha1.CryptoTracker{}
	if err := r.Get(ctx, req.NamespacedName, tracker); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	//build environment variables
	enVars := []corev1.EnvVar{
		{Name: "SYMBOL", Value: tracker.Spec.Symbol},
	}

	//check for db ref
	if tracker.Spec.DB != "" {
		db := &databasev1alpha1.CryptoDatabase{}
		dbKey := client.ObjectKey{Name: tracker.Spec.DB, Namespace: tracker.Namespace}

		if err := r.Get(ctx, dbKey, db); err == nil {
			//found it! inject connection details
			//db controller names the service == db name. So here it's enough to refer to db name
			l.Info("connecting to database", "db", db.Name)
			enVars = append(enVars,
				corev1.EnvVar{Name: "DB_HOST", Value: db.Name},
				corev1.EnvVar{Name: "DB_PORT", Value: "5432"},
				corev1.EnvVar{Name: "DB_USER", Value: "user"},
				corev1.EnvVar{Name: "DB_PASSWORD", Value: "password"},
				corev1.EnvVar{Name: "DB_NAME", Value: "crypto"},
			)
		} else {
			l.Error(err, "database referenced but not found", "db_name", tracker.Spec.DB)
			//graceful handling, we couldn't create db, we run it stateless
		}
	}

	//check for cache ref
	if tracker.Spec.Cache != "" {
		cache := &databasev1alpha1.CryptoCache{}
		cacheKey := client.ObjectKey{Name: tracker.Spec.Cache, Namespace: tracker.Namespace}

		if err := r.Get(ctx, cacheKey, cache); err == nil {
			//found the cache, inject redis address
			l.Info("connecting to cache", "cache", cache.Name)
			enVars = append(enVars,
				corev1.EnvVar{Name: "REDIS_ADDR", Value: cache.Name + ":6379"},
			)
		} else {
			l.Error(err, "cache referenced but not found", "cache_name", tracker.Spec.Cache)
		}
	}

	//create the application deployment
	dep := r.desiredDeployment(tracker, enVars)

	applyOpts := []client.PatchOption{client.ForceOwnership, client.FieldOwner("crypto-operator")}
	if err := r.Patch(ctx, dep, client.Apply, applyOpts...); err != nil {
		l.Error(err, "failed to apply deployment")
		return ctrl.Result{}, err
	}

	svc := r.desiredService(tracker)
	if err := r.Patch(ctx, svc, client.Apply, applyOpts...); err != nil {
		l.Error(err, "failed to apply service")
		return ctrl.Result{}, err
	}

	l.Info("Reconciled CryptoTracker", "name", tracker.Name)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CryptoTrackerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha1.CryptoTracker{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

func (r *CryptoTrackerReconciler) desiredDeployment(tracker *appv1alpha1.CryptoTracker, envs []corev1.EnvVar) *appsv1.Deployment {
	commonLabels := map[string]string{"app": "crypto-tracker", "instance": tracker.Name}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      tracker.Name,
			Namespace: tracker.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: commonLabels},
			Replicas: int32Ptr(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: commonLabels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "manager",
							Image: tracker.Spec.Image,
							Env:   envs,
							Ports: []corev1.ContainerPort{{ContainerPort: 3000}},
						},
					},
				},
			},
		},
	}
}

func (r *CryptoTrackerReconciler) desiredService(tracker *appv1alpha1.CryptoTracker) *corev1.Service {
	commonLabels := map[string]string{"app": "crypto-tracker", "instance": tracker.Name}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      tracker.Name,
			Namespace: tracker.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: commonLabels,
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(3000),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}

func int32Ptr(i int32) *int32 { return &i }
