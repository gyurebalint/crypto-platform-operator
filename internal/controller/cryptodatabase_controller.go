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

// CryptoDatabaseReconciler reconciles a CryptoDatabase object
type CryptoDatabaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=database.crypto.com,resources=cryptodatabases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=database.crypto.com,resources=cryptodatabases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=database.crypto.com,resources=cryptodatabases/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CryptoDatabase object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *CryptoDatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	// fetch the CryptoDatbase instance
	db := &databasev1alpha1.CryptoDatabase{}
	if err := r.Get(ctx, req.NamespacedName, db); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// define the stateful set
	sts := r.desiredStatefulSet(db)

	//apply stateful set
	applyOpt := []client.PatchOption{client.ForceOwnership, client.FieldOwner("crypto-operator")}
	if err := r.Patch(ctx, sts, client.Apply, applyOpt...); err != nil {
		l.Error(err, "Failed to apply StatefulSet")
		return ctrl.Result{}, err
	}

	svc := r.desiredService(db)

	if err := r.Patch(ctx, svc, client.Apply, applyOpt...); err != nil {
		l.Error(err, "Failed to apply Service")
		return ctrl.Result{}, err
	}

	l.Info("Reconciled CryptoDatabase", "name", db.Name)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CryptoDatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1alpha1.CryptoDatabase{}).
		Complete(r)
}

// desiredStatefulSet returns the minimal StatefulSet for Postgres
func (r *CryptoDatabaseReconciler) desiredStatefulSet(db *databasev1alpha1.CryptoDatabase) *appsv1.StatefulSet {
	commonLabels := map[string]string{"app": "crypto-database", "instance": db.Name}

	return &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      db.Name,
			Namespace: db.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: commonLabels,
			},
			ServiceName: db.Name,
			Replicas:    int32Ptr(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: commonLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "postgres",
							Image: "postgres:15-alpine",
							Env: []corev1.EnvVar{
								{Name: "POSTGRES_USER", Value: "user"},
								{Name: "POSTGRES_PASSWORD", Value: "password"},
								{Name: "POSTGRES_DB", Value: "crypto"},
							},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 5432, Name: "postgres"},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "data", MountPath: "/var/lib/postgresql/data"},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "data"},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse(db.Spec.StorageSize),
							},
						},
					},
				},
			},
		},
	}
}

// desiredService returns the Service to expose Postgres
func (r *CryptoDatabaseReconciler) desiredService(db *databasev1alpha1.CryptoDatabase) *corev1.Service {
	commonLabels := map[string]string{"app": "crypto-database", "instance": db.Name}

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      db.Name,
			Namespace: db.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: commonLabels,
			Ports: []corev1.ServicePort{
				{
					Port:       5432,
					TargetPort: intstr.FromInt(5432),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}

func int32Ptr(i int32) *int32 { return &i }
