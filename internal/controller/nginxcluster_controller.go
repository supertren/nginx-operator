package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1alpha1 "github.com/example/nginx-operator/api/v1alpha1"
)

type NginxClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//nolint:revive
//+kubebuilder:rbac:groups=apps.example.com,resources=nginxclusters,verbs=get;list;watch;create;update;patch;delete
//nolint:revive
//+kubebuilder:rbac:groups=apps.example.com,resources=nginxclusters/status,verbs=get;update;patch
//nolint:revive
//+kubebuilder:rbac:groups=apps.example.com,resources=nginxclusters/finalizers,verbs=update
//nolint:revive
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//nolint:revive
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *NginxClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. Obtener el objeto NginxCluster
	nginx := &appsv1alpha1.NginxCluster{}
	if err := r.Get(ctx, req.NamespacedName, nginx); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Reconciling NginxCluster", "name", nginx.Name, "replicas", nginx.Spec.Replicas)

	// 2. Reconciliar el Deployment
	if err := r.reconcileDeployment(ctx, nginx); err != nil {
		return ctrl.Result{}, err
	}

	// 3. Reconciliar el Service
	if err := r.reconcileService(ctx, nginx); err != nil {
		return ctrl.Result{}, err
	}

	// 4. Actualizar el Status
	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: nginx.Name, Namespace: nginx.Namespace}, deployment); err == nil {
		nginx.Status.AvailableReplicas = deployment.Status.AvailableReplicas
		if err := r.Status().Update(ctx, nginx); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *NginxClusterReconciler) reconcileDeployment(ctx context.Context, nginx *appsv1alpha1.NginxCluster) error {
	logger := log.FromContext(ctx)
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: nginx.Name, Namespace: nginx.Namespace}, deployment)

	if errors.IsNotFound(err) {
		// Crear el Deployment
		dep := r.buildDeployment(nginx)
		logger.Info("Creating Deployment", "name", dep.Name)
		return r.Create(ctx, dep)
	}
	if err != nil {
		return err
	}

	// Actualizar replicas si no coinciden
	if nginx.Spec.Replicas != nil && (deployment.Spec.Replicas == nil || *deployment.Spec.Replicas != *nginx.Spec.Replicas) {
		deployment.Spec.Replicas = nginx.Spec.Replicas
		logger.Info("Updating Deployment replicas", "replicas", *nginx.Spec.Replicas)
		return r.Update(ctx, deployment)
	}

	return nil
}

func (r *NginxClusterReconciler) reconcileService(ctx context.Context, nginx *appsv1alpha1.NginxCluster) error {
	logger := log.FromContext(ctx)
	svc := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: nginx.Name, Namespace: nginx.Namespace}, svc)

	if errors.IsNotFound(err) {
		service := r.buildService(nginx)
		logger.Info("Creating Service", "name", service.Name)
		return r.Create(ctx, service)
	}

	return err
}

func (r *NginxClusterReconciler) buildDeployment(nginx *appsv1alpha1.NginxCluster) *appsv1.Deployment {
	replicas := nginx.Spec.Replicas
	labels := map[string]string{"app": nginx.Name}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nginx.Name,
			Namespace: nginx.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "nginx",
						Image: "docker.io/nginxinc/nginx-unprivileged:latest",
						Ports: []corev1.ContainerPort{{ContainerPort: 8080}},
					}},
				},
			},
		},
	}
	_ = ctrl.SetControllerReference(nginx, dep, r.Scheme)
	return dep
}

func (r *NginxClusterReconciler) buildService(nginx *appsv1alpha1.NginxCluster) *corev1.Service {
	labels := map[string]string{"app": nginx.Name}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nginx.Name,
			Namespace: nginx.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Port:     8080,
				Protocol: corev1.ProtocolTCP,
			}},
		},
	}
	_ = ctrl.SetControllerReference(nginx, svc, r.Scheme)
	return svc
}

func (r *NginxClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1alpha1.NginxCluster{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
