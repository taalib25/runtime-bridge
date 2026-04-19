package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	runtimev1alpha1 "github.com/hermeshq/hermes-runtime-operator/api/v1alpha1"
	"github.com/hermeshq/hermes-runtime-operator/internal/runtimeutil"
)

type RuntimeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=hermes.hermeshq.net,resources=runtimes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=hermes.hermeshq.net,resources=runtimes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=hermes.hermeshq.net,resources=runtimes/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps;persistentvolumeclaims;secrets;services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *RuntimeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	rt := &runtimev1alpha1.Runtime{}
	if err := r.Get(ctx, req.NamespacedName, rt); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !rt.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	if err := r.reconcile(ctx, rt); err != nil {
		if apierrors.IsConflict(err) {
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}
		logger.Error(err, "Failed to reconcile Runtime", "name", rt.Name, "namespace", rt.Namespace)
		_ = r.updateStatus(ctx, rt, runtimev1alpha1.RuntimePhaseFailed, err.Error(), 0, 1)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: runtimeutil.DeploymentName(rt.Spec.RuntimeID), Namespace: rt.Namespace}, deployment); err != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	if deployment.Status.ReadyReplicas < 1 {
		_ = r.updateStatus(ctx, rt, runtimev1alpha1.RuntimePhaseCreating, "Runtime is starting", deployment.Status.ReadyReplicas, 1)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	if err := r.updateStatus(ctx, rt, runtimev1alpha1.RuntimePhaseRunning, "Runtime is ready", deployment.Status.ReadyReplicas, 1); err != nil {
		if apierrors.IsConflict(err) {
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RuntimeReconciler) reconcile(ctx context.Context, rt *runtimev1alpha1.Runtime) error {
	if err := r.reconcileConfigMap(ctx, rt); err != nil {
		return err
	}
	if err := r.reconcilePVC(ctx, rt); err != nil {
		return err
	}
	if err := r.reconcileDeployment(ctx, rt); err != nil {
		return err
	}
	if err := r.reconcileService(ctx, rt); err != nil {
		return err
	}
	if err := r.reconcileIngress(ctx, rt); err != nil {
		return err
	}
	return nil
}

func (r *RuntimeReconciler) reconcileConfigMap(ctx context.Context, rt *runtimev1alpha1.Runtime) error {
	configYAML, err := runtimeutil.RenderConfigYAML(rt.Spec)
	if err != nil {
		return err
	}
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: runtimeutil.ConfigMapName(rt.Spec.RuntimeID), Namespace: rt.Namespace}}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Labels = runtimeutil.Labels(rt)
		cm.Data = map[string]string{
			"config.yaml": configYAML,
			"SOUL.md":     rt.Spec.Soul,
		}
		return controllerutil.SetControllerReference(rt, cm, r.Scheme)
	})
	return err
}

func (r *RuntimeReconciler) reconcilePVC(ctx context.Context, rt *runtimev1alpha1.Runtime) error {
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: runtimeutil.PVCName(rt.Spec.RuntimeID), Namespace: rt.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, pvc, func() error {
		pvc.Labels = runtimeutil.Labels(rt)
		if pvc.Spec.AccessModes == nil {
			pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
		}
		pvc.Spec.Resources.Requests = corev1.ResourceList{
			corev1.ResourceStorage: apiresource.MustParse(fmt.Sprintf("%dGi", rt.Spec.Storage.SizeGi)),
		}
		if rt.Spec.Storage.StorageClassName != "" {
			pvc.Spec.StorageClassName = ptr.To(rt.Spec.Storage.StorageClassName)
		}
		return controllerutil.SetControllerReference(rt, pvc, r.Scheme)
	})
	if apierrors.IsInvalid(err) || apierrors.IsForbidden(err) {
		return err
	}
	return err
}

func (r *RuntimeReconciler) reconcileDeployment(ctx context.Context, rt *runtimev1alpha1.Runtime) error {
	labels := runtimeutil.Labels(rt)
	configYAML, err := runtimeutil.RenderConfigYAML(rt.Spec)
	if err != nil {
		return err
	}
	configChecksum := runtimeutil.ConfigChecksum(configYAML, rt.Spec.Soul)
	managedSecret, err := r.getManagedSecret(ctx, rt)
	if err != nil {
		return err
	}
	secretChecksum := runtimeutil.SecretChecksum(managedSecret)
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: runtimeutil.DeploymentName(rt.Spec.RuntimeID), Namespace: rt.Namespace}}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, dep, func() error {
		dep.Labels = labels
		dep.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
		dep.Spec.Replicas = ptr.To(int32(1))
		dep.Spec.Strategy = appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
		dep.Spec.Template.ObjectMeta.Labels = labels
		if dep.Spec.Template.ObjectMeta.Annotations == nil {
			dep.Spec.Template.ObjectMeta.Annotations = map[string]string{}
		}
		dep.Spec.Template.ObjectMeta.Annotations["checksum/config"] = configChecksum
		dep.Spec.Template.ObjectMeta.Annotations["checksum/secret"] = secretChecksum
		dep.Spec.Template.Spec.AutomountServiceAccountToken = ptr.To(false)
		dep.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
			RunAsNonRoot:        ptr.To(true),
			RunAsUser:           ptr.To(int64(1000)),
			RunAsGroup:          ptr.To(int64(1000)),
			FSGroup:             ptr.To(int64(1000)),
			FSGroupChangePolicy: ptr.To(corev1.FSGroupChangeOnRootMismatch),
			SeccompProfile:      &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
		}
		dep.Spec.Template.Spec.InitContainers = []corev1.Container{bootstrapContainer(rt)}
		dep.Spec.Template.Spec.Containers = []corev1.Container{runtimeContainer(rt)}
		dep.Spec.Template.Spec.Volumes = runtimeVolumes(rt)
		return controllerutil.SetControllerReference(rt, dep, r.Scheme)
	})
	return err
}

func (r *RuntimeReconciler) reconcileService(ctx context.Context, rt *runtimev1alpha1.Runtime) error {
	labels := runtimeutil.Labels(rt)
	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: runtimeutil.ServiceName(rt.Spec.RuntimeID), Namespace: rt.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
		svc.Labels = labels
		svc.Spec.Selector = labels
		svc.Spec.Type = corev1.ServiceTypeClusterIP
		svc.Spec.Ports = []corev1.ServicePort{{Name: "api-server", Port: 8642, TargetPort: intstrFromInt(8642), Protocol: corev1.ProtocolTCP}}
		return controllerutil.SetControllerReference(rt, svc, r.Scheme)
	})
	return err
}

func (r *RuntimeReconciler) reconcileIngress(ctx context.Context, rt *runtimev1alpha1.Runtime) error {
	host, err := runtimeutil.Host(rt.Spec)
	if err != nil {
		return err
	}
	path, err := runtimeutil.Path(rt.Spec)
	if err != nil {
		return err
	}
	labels := runtimeutil.Labels(rt)
	ing := &networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: runtimeutil.IngressName(rt.Spec.RuntimeID), Namespace: rt.Namespace}}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, ing, func() error {
		ing.Labels = labels
		if ing.Annotations == nil {
			ing.Annotations = map[string]string{}
		}
		ing.Annotations["traefik.ingress.kubernetes.io/router.entrypoints"] = "web,websecure"
		className := rt.Spec.Network.IngressClassName
		if className == "" {
			className = "traefik"
		}
		ing.Spec.IngressClassName = ptr.To(className)
		ing.Spec.Rules = []networkingv1.IngressRule{{
			Host: host,
			IngressRuleValue: networkingv1.IngressRuleValue{
				HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{
					Path:     path,
					PathType: ptr.To(networkingv1.PathTypePrefix),
					Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{
						Name: runtimeutil.ServiceName(rt.Spec.RuntimeID),
						Port: networkingv1.ServiceBackendPort{Number: 8642},
					}},
				}}},
			},
		}}
		return controllerutil.SetControllerReference(rt, ing, r.Scheme)
	})
	return err
}

func (r *RuntimeReconciler) updateStatus(ctx context.Context, rt *runtimev1alpha1.Runtime, phase runtimev1alpha1.RuntimePhase, message string, readyReplicas int32, desiredReplicas int32) error {
	current := &runtimev1alpha1.Runtime{}
	if err := r.Get(ctx, types.NamespacedName{Name: rt.Name, Namespace: rt.Namespace}, current); err != nil {
		return err
	}
	url, err := runtimeutil.URL(current.Spec)
	if err != nil {
		return err
	}
	current.Status.Phase = phase
	current.Status.URL = url
	current.Status.Namespace = current.Namespace
	current.Status.DeploymentName = runtimeutil.DeploymentName(current.Spec.RuntimeID)
	current.Status.ServiceName = runtimeutil.ServiceName(current.Spec.RuntimeID)
	current.Status.IngressName = runtimeutil.IngressName(current.Spec.RuntimeID)
	current.Status.ReadyReplicas = readyReplicas
	current.Status.DesiredReplicas = desiredReplicas
	current.Status.Message = message
	current.Status.LastReconciledAt = &metav1.Time{Time: time.Now().UTC()}
	current.Status.ObservedGeneration = current.Generation
	return r.Status().Update(ctx, current)
}

func (r *RuntimeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&runtimev1alpha1.Runtime{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Owns(&networkingv1.Ingress{}).
		Named("runtime").
		Complete(r)
}

func (r *RuntimeReconciler) getManagedSecret(ctx context.Context, rt *runtimev1alpha1.Runtime) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: runtimeutil.ManagedSecretName(rt.Spec.RuntimeID), Namespace: rt.Namespace}, secret)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return secret, nil
}

func bootstrapContainer(rt *runtimev1alpha1.Runtime) corev1.Container {
	return corev1.Container{
		Name:            "bootstrap-config",
		Image:           rt.Spec.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/bin/sh", "-ec"},
		Args: []string{`set -eu
mkdir -p /opt/data/home
cp /bootstrap/config.yaml /opt/data/config.yaml
cp /bootstrap/SOUL.md /opt/data/SOUL.md`},
		Env: []corev1.EnvVar{{Name: "HERMES_HOME", Value: "/opt/data"}},
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
			Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
			ReadOnlyRootFilesystem:   ptr.To(true),
			SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "data", MountPath: "/opt/data"},
			{Name: "bootstrap", MountPath: "/bootstrap", ReadOnly: true},
			{Name: "tmp", MountPath: "/tmp"},
		},
	}
}

func runtimeContainer(rt *runtimev1alpha1.Runtime) corev1.Container {
	limitsCPU := apiresource.MustParse(fmt.Sprintf("%dm", rt.Spec.Plan.CPUMillis))
	limitsMemory := apiresource.MustParse(fmt.Sprintf("%dMi", rt.Spec.Plan.MemoryMi))
	requestsCPU := apiresource.MustParse(fmt.Sprintf("%dm", maxInt64(50, rt.Spec.Plan.CPUMillis/2)))
	requestsMemory := apiresource.MustParse(fmt.Sprintf("%dMi", maxInt64(128, rt.Spec.Plan.MemoryMi/2)))

	env := []corev1.EnvVar{
		{Name: "HERMES_HOME", Value: "/opt/data"},
		{Name: "HOME", Value: "/opt/data/home"},
		{Name: "GATEWAY_ALLOW_ALL_USERS", Value: "false"},
		{Name: "WEB_TOOLS_DEBUG", Value: "false"},
		{Name: "VISION_TOOLS_DEBUG", Value: "false"},
		{Name: "MOA_TOOLS_DEBUG", Value: "false"},
		{Name: "IMAGE_TOOLS_DEBUG", Value: "false"},
		{Name: "HERMES_HUMAN_DELAY_MODE", Value: "off"},
		{Name: "API_SERVER_ENABLED", Value: "true"},
		{Name: "API_SERVER_HOST", Value: "0.0.0.0"},
		{Name: "API_SERVER_PORT", Value: "8642"},
		{Name: "API_SERVER_MODEL_NAME", Value: "hermes-agent"},
	}
	for _, item := range rt.Spec.Env {
		env = append(env, corev1.EnvVar{Name: item.Name, Value: item.Value})
	}

	envFrom := make([]corev1.EnvFromSource, 0, len(rt.Spec.EnvFromSecrets))
	for _, item := range rt.Spec.EnvFromSecrets {
		envFrom = append(envFrom, corev1.EnvFromSource{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: item.Name}}})
	}

	return corev1.Container{
		Name:            "hermes-agent",
		Image:           rt.Spec.Image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args:            []string{"gateway", "run"},
		WorkingDir:      "/opt/hermes",
		Ports:           []corev1.ContainerPort{{Name: "api-server", ContainerPort: 8642, Protocol: corev1.ProtocolTCP}},
		Env:             env,
		EnvFrom:         envFrom,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceCPU: requestsCPU, corev1.ResourceMemory: requestsMemory},
			Limits:   corev1.ResourceList{corev1.ResourceCPU: limitsCPU, corev1.ResourceMemory: limitsMemory},
		},
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
			Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
			ReadOnlyRootFilesystem:   ptr.To(true),
			SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
		},
		VolumeMounts: []corev1.VolumeMount{
			{Name: "data", MountPath: "/opt/data"},
			{Name: "tmp", MountPath: "/tmp"},
			{Name: "dshm", MountPath: "/dev/shm"},
		},
		LivenessProbe:  &corev1.Probe{ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/health", Port: intstrFromInt(8642)}}, InitialDelaySeconds: 30, PeriodSeconds: 30},
		ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/health", Port: intstrFromInt(8642)}}, InitialDelaySeconds: 10, PeriodSeconds: 10},
	}
}

func runtimeVolumes(rt *runtimev1alpha1.Runtime) []corev1.Volume {
	return []corev1.Volume{
		{Name: "data", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: runtimeutil.PVCName(rt.Spec.RuntimeID)}}},
		{Name: "bootstrap", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: runtimeutil.ConfigMapName(rt.Spec.RuntimeID)}}}},
		{Name: "tmp", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		{Name: "dshm", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{Medium: corev1.StorageMediumMemory, SizeLimit: resourceQuantityPtr(apiresource.MustParse("1Gi"))}}},
	}
}

func intstrFromInt(v int) intstr.IntOrString {
	return intstr.FromInt(v)
}

func resourceQuantityPtr(q apiresource.Quantity) *apiresource.Quantity {
	return &q
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
