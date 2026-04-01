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
	"fmt"
	"reflect"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	appsexamplecomv1alpha1 "apps.example.com/vibe-operator/api/v1alpha1"
)

// VibeAppReconciler 是 VibeApp 自定义资源的控制器（Reconciler）
// 它负责监视 VibeApp 资源的变化，并协调创建/更新/删除相关的 Deployment 和 Service
//
// 主要职责：
//  1. 监听 VibeApp 资源的创建、更新、删除事件
//  2. 根据 VibeApp Spec 创建对应的 Deployment 和 Service
//  3. 设置 OwnerReference 确保级联删除
//  4. 实现自愈逻辑，防止副本数被手动修改为小于 2
//  5. 更新 VibeApp 的状态条件（Conditions）
type VibeAppReconciler struct {
	// Client 是 Kubernetes 的 clientset，用于与 API Server 交互
	client.Client
	// Scheme 用于序列化/反序列化 Kubernetes 对象
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=apps.example.com,resources=vibeapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.example.com,resources=vibeapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.example.com,resources=vibeapps/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch

// Reconcile 是控制器的主循环，由 controller-runtime 在以下情况调用：
//   - VibeApp 资源被创建、更新或删除时
//   - 关联的 Deployment 或 Service 发生变化时（通过 Watches 机制）
//
// 函数遵循标准的 reconcile 模式：
//  1. 读取主资源（VibeApp）
//  2. 处理删除逻辑
//  3. 协调关联的子资源（Deployment、Service）
//  4. 更新主资源状态
//
// 返回值：
//   - ctrl.Result: 控制下一次 reconcile 的时间间隔（用于重试或速率限制）
//   - error: 如果出错，控制器会自动重试（除非是 IsNotFound 等特定错误）
func (r *VibeAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log = log.WithValues("vibeapp", req.NamespacedName)

	// 1. 获取 VibeApp 实例
	// 使用 structured logging 记录 reconcile 开始
	log.Info("Starting reconciliation")
	vibesapp := &appsexamplecomv1alpha1.VibeApp{}
	if err := r.Get(ctx, req.NamespacedName, vibesapp); err != nil {
		// 如果资源不存在，可能是已被删除，直接返回不再处理
		if errors.IsNotFound(err) {
			log.Info("VibeApp resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// 其他错误记录并返回，触发重试
		log.Error(err, "Failed to get VibeApp")
		return ctrl.Result{}, err
	}

	// 2. 处理资源删除
	// 如果 DeletionTimestamp 被设置，说明资源正在被删除
	// 由于我们设置了 OwnerReference，Kubernetes GC 会自动清理 owned resources
	// 因此这里只需要简单返回，不需要额外清理逻辑
	if !vibesapp.DeletionTimestamp.IsZero() {
		log.Info("VibeApp is being deleted. DeletionTimestamp is set")
		return ctrl.Result{}, nil
	}

	// 3. 确保 Deployment 存在且符合期望状态
	// 构建期望的 Deployment 对象（desired state）
	desiredDeployment := r.deploymentForVibeApp(vibesapp)
	// 调和 Deployment：创建、更新或修复
	if err := r.reconcileDeployment(ctx, vibesapp, desiredDeployment); err != nil {
		// Deployment 失败只记录错误，不中断后续 Service 处理
		// 这样可以确保部分失败时至少其他资源能被正确创建/更新
		log.Error(err, "Failed to reconcile Deployment")
		// TODO: 可选 - 更新状态条件为 Degraded，记录失败原因
		// r.updateStatusCondition(ctx, vibesapp, "Degraded", "DeploymentReconciliationFailed", err.Error())
	}

	// 4. 确保 Service 存在且符合期望状态
	desiredService := r.serviceForVibeApp(vibesapp)
	if err := r.reconcileService(ctx, vibesapp, desiredService); err != nil {
		log.Error(err, "Failed to reconcile Service")
		// TODO: 可选 - 更新状态条件为 Degraded
	}

	// 5. 更新 VibeApp 状态
	// 更新 ObservedGeneration 和 Available condition
	r.updateStatus(ctx, vibesapp)

	log.Info("Reconciliation completed successfully")
	return ctrl.Result{}, nil
}

// deploymentForVibeApp 根据 VibeApp Spec 构建期望的 Deployment 对象
// 这是一个声明式的构建函数，返回的 Deployment 不包含运行时字段（如 Status）
func (r *VibeAppReconciler) deploymentForVibeApp(vibesapp *appsexamplecomv1alpha1.VibeApp) *appsv1.Deployment {
	// 确定副本数：如果小于 2 则使用 2（这也用于自愈逻辑）
	replicas := vibesapp.Spec.Replicas
	if replicas < 2 {
		replicas = 2
	}

	// 获取标准标签（用于资源标识和 selector）
	labels := r.labelsForVibeApp(vibesapp)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vibesapp.Name,      // Deployment 名称与 VibeApp 相同
			Namespace: vibesapp.Namespace, // 与 VibeApp 同 namespace
			Labels:    labels,             // 打上标准标签
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas, // 副本数（ptr 类型）
			Selector: &metav1.LabelSelector{
				// selector 必须匹配 pod template 的标签
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels, // Pod 也打上相同标签
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "vibeapp", // 容器名称固定
							Image: vibesapp.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: vibesapp.Spec.Port,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							// 挂载存储卷
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "storage", // 对应 volumes[0].Name
									MountPath: "/data",   // 容器内挂载路径（固定）
								},
							},
							// 存活探针：用于 Kubernetes 判断容器是否健康
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: vibesapp.Spec.HealthCheckPath, // 使用用户指定的健康检查路径
										Port: intstr.FromInt(int(vibesapp.Spec.Port)),
									},
								},
								InitialDelaySeconds: 30, // 容器启动后 30 秒开始探测
								PeriodSeconds:       10, // 每 10 秒探测一次
								TimeoutSeconds:      5,  // 超时时间 5 秒
								FailureThreshold:    3,  // 连续失败 3 次标记为不健康
							},
							// 就绪探针：用于 Kubernetes 判断容器是否准备好接收流量
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: vibesapp.Spec.HealthCheckPath,
										Port: intstr.FromInt(int(vibesapp.Spec.Port)),
									},
								},
								InitialDelaySeconds: 5,  // 容器启动后 5 秒开始探测
								PeriodSeconds:       10, // 每 10 秒探测一次
								TimeoutSeconds:      5,  // 超时时间 5 秒
								FailureThreshold:    3,  // 连续失败 3 次标记为未就绪
							},
						},
					},
					// HostPath 卷：将宿主机目录挂载到容器
					Volumes: []corev1.Volume{
						{
							Name: "storage",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: vibesapp.Spec.StoragePath, // 使用用户指定的宿主机路径
								},
							},
						},
					},
				},
			},
		},
	}

	// 关键：设置 OwnerReference 以实现级联删除
	// 当 VibeApp 被删除时，Kubernetes 会自动删除所有 owned resources
	controllerutil.SetControllerReference(vibesapp, deployment, r.Scheme)

	return deployment
}

// reconcileDeployment 调和 Deployment：创建新资源或更新现有资源
//   - 如果 Deployment 不存在，则创建
//   - 如果存在但副本数小于期望值（自愈逻辑），则通过 Patch 修复
//   - 如果存在且需要更新，则更新（通过 deploymentNeedsUpdate 判断）
//
// 错误处理策略：
//   - 创建/更新失败：返回包装后的错误，触发 reconcile 重试
//   - 冲突（IsConflict）：返回错误让控制器重试（基于 exponential backoff）
//   - 其他错误：记录日志并返回
func (r *VibeAppReconciler) reconcileDeployment(ctx context.Context, vibesapp *appsexamplecomv1alpha1.VibeApp, desired *appsv1.Deployment) error {
	log := logf.FromContext(ctx).WithValues("deployment", desired.Name, "namespace", desired.Namespace)

	// 查找是否已存在
	existing := &appsv1.Deployment{}
	namespacedName := types.NamespacedName{
		Namespace: desired.Namespace,
		Name:      desired.Name,
	}
	err := r.Get(ctx, namespacedName, existing)

	if err != nil && errors.IsNotFound(err) {
		// 创建新 Deployment
		log.Info("Creating Deployment",
			"replicas", desired.Spec.Replicas,
			"image", desired.Spec.Template.Spec.Containers[0].Image,
			"port", desired.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
		if err := r.Create(ctx, desired); err != nil {
			log.Error(err, "Failed to create Deployment")
			// 使用 %w 包装错误，便于调用者检查错误类型
			return fmt.Errorf("failed to create Deployment: %w", err)
		}
		log.Info("Deployment created successfully", "name", desired.Name)
		return nil
	} else if err != nil {
		// 获取现有资源失败（非 NotFound）
		log.Error(err, "Failed to get existing Deployment")
		return fmt.Errorf("failed to get Deployment: %w", err)
	}

	// 自愈逻辑：检查副本数是否小于期望值
	// 期望副本数 = max(vibesapp.Spec.Replicas, 2)
	expectedReplicas := int32(2)
	if vibesapp.Spec.Replicas > expectedReplicas {
		expectedReplicas = vibesapp.Spec.Replicas
	}
	if *existing.Spec.Replicas < expectedReplicas {
		log.Info("Fixing Deployment replicas (self-healing)",
			"current", *existing.Spec.Replicas,
			"expected", expectedReplicas)
		// 使用 strategic merge patch 只更新 replicas 字段，避免覆盖其他修改
		patch := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Replicas: &expectedReplicas,
			},
		}
		if err := r.Patch(ctx, patch, client.MergeFrom(existing)); err != nil {
			log.Error(err, "Failed to patch Deployment replicas")
			return fmt.Errorf("failed to patch Deployment replicas: %w", err)
		}
		log.Info("Deployment replicas fixed successfully", "replicas", expectedReplicas)
		return nil
	}

	// 检查是否有实际变化（避免不必要的更新）
	if r.deploymentNeedsUpdate(existing, desired) {
		// 更新 Deployment（完整替换 Spec）
		existing.Spec.Replicas = desired.Spec.Replicas
		existing.Spec.Selector = desired.Spec.Selector
		existing.Spec.Template = desired.Spec.Template

		log.Info("Updating Deployment",
			"replicas", desired.Spec.Replicas,
			"image", desired.Spec.Template.Spec.Containers[0].Image)
		if err := r.Update(ctx, existing); err != nil {
			// Kubernetes 冲突检测：如果 resourceVersion 不匹配会返回冲突错误
			// 返回错误让控制器重试（controller-runtime 会自动处理 retry）
			if errors.IsConflict(err) {
				log.Info("Deployment update conflict, will retry")
				return err
			}
			log.Error(err, "Failed to update Deployment")
			return fmt.Errorf("failed to update Deployment: %w", err)
		}
		log.Info("Deployment updated successfully", "name", desired.Name)
	} else {
		// Verbose 级别日志，只有设置 -v=2 时才显示
		log.V(1).Info("Deployment is up to date, skipping update")
	}

	return nil
}

// deploymentNeedsUpdate 对比现有和期望的 Deployment，判断是否需要更新
// 这是一个简化版的对比，只检查关键字段的变化
// 返回 true 如果检测到任何重要变化
func (r *VibeAppReconciler) deploymentNeedsUpdate(existing, desired *appsv1.Deployment) bool {
	// 对比副本数
	if *existing.Spec.Replicas != *desired.Spec.Replicas {
		return true
	}

	// 对比容器镜像
	existingImage := existing.Spec.Template.Spec.Containers[0].Image
	desiredImage := desired.Spec.Template.Spec.Containers[0].Image
	if existingImage != desiredImage {
		return true
	}

	// 对比端口数量（简单检查容器端口配置是否变化）
	if len(existing.Spec.Template.Spec.Containers[0].Ports) != len(desired.Spec.Template.Spec.Containers[0].Ports) {
		return true
	}

	// 对比卷数量
	if len(existing.Spec.Template.Spec.Volumes) != len(desired.Spec.Template.Spec.Volumes) {
		return true
	}

	// 对比标签选择器（必须匹配才能正确筛选 Pods）
	if !reflect.DeepEqual(existing.Spec.Selector, desired.Spec.Selector) {
		return true
	}

	// 对比 Pod 模板标签
	if !reflect.DeepEqual(existing.Spec.Template.Labels, desired.Spec.Template.Labels) {
		return true
	}

	// 所有关键字段都相同，不需要更新
	return false
}

// serviceForVibeApp 根据 VibeApp Spec 构建期望的 Service 对象
func (r *VibeAppReconciler) serviceForVibeApp(vibesapp *appsexamplecomv1alpha1.VibeApp) *corev1.Service {
	labels := r.labelsForVibeApp(vibesapp)
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vibesapp.Name,
			Namespace: vibesapp.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			// selector 用于选择属于该 Service 的 Pod
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       vibesapp.Spec.Port,                      // Service 端口
					TargetPort: intstr.FromInt(int(vibesapp.Spec.Port)), // 容器端口（与 Pod 一致）
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP, // 默认类型：集群内部访问
		},
	}

	// 设置 OwnerReference 以确保级联删除
	controllerutil.SetControllerReference(vibesapp, service, r.Scheme)

	return service
}

// reconcileService 调和 Service：创建新资源或更新现有资源
// 逻辑与 reconcileDeployment 类似，但处理的是 Service 资源
func (r *VibeAppReconciler) reconcileService(ctx context.Context, vibesapp *appsexamplecomv1alpha1.VibeApp, desired *corev1.Service) error {
	log := logf.FromContext(ctx).WithValues("service", desired.Name, "namespace", desired.Namespace)

	existing := &corev1.Service{}
	namespacedName := types.NamespacedName{
		Namespace: desired.Namespace,
		Name:      desired.Name,
	}
	err := r.Get(ctx, namespacedName, existing)

	if err != nil && errors.IsNotFound(err) {
		// 创建新 Service
		log.Info("Creating Service",
			"type", desired.Spec.Type,
			"port", desired.Spec.Ports[0].Port,
			"targetPort", desired.Spec.Ports[0].TargetPort.IntValue())
		if err := r.Create(ctx, desired); err != nil {
			log.Error(err, "Failed to create Service")
			return fmt.Errorf("failed to create Service: %w", err)
		}
		log.Info("Service created successfully", "name", desired.Name)
		return nil
	} else if err != nil {
		log.Error(err, "Failed to get existing Service")
		return fmt.Errorf("failed to get Service: %w", err)
	}

	// 检查是否需要更新
	if r.serviceNeedsUpdate(existing, desired) {
		existing.Spec.Selector = desired.Spec.Selector
		existing.Spec.Ports = desired.Spec.Ports
		existing.Spec.Type = desired.Spec.Type

		log.Info("Updating Service",
			"type", desired.Spec.Type,
			"port", desired.Spec.Ports[0].Port)
		if err := r.Update(ctx, existing); err != nil {
			// 处理资源冲突
			if errors.IsConflict(err) {
				log.Info("Service update conflict, will retry")
				return err
			}
			log.Error(err, "Failed to update Service")
			return fmt.Errorf("failed to update Service: %w", err)
		}
		log.Info("Service updated successfully", "name", desired.Name)
	} else {
		log.V(1).Info("Service is up to date, skipping update")
	}

	return nil
}

// serviceNeedsUpdate 检查 Service 是否需要更新
func (r *VibeAppReconciler) serviceNeedsUpdate(existing, desired *corev1.Service) bool {
	// 对比 Service 类型（如 ClusterIP、NodePort 等）
	if existing.Spec.Type != desired.Spec.Type {
		return true
	}

	// 对比端口数量
	if len(existing.Spec.Ports) != len(desired.Spec.Ports) {
		return true
	}

	// 对比 selector 字段数
	if len(existing.Spec.Selector) != len(desired.Spec.Selector) {
		return true
	}

	// 逐键对比 selector
	for k, v := range desired.Spec.Selector {
		if existing.Spec.Selector[k] != v {
			return true
		}
	}

	// 对比端口详情（端口号、目标端口、协议）
	if len(existing.Spec.Ports) > 0 && len(desired.Spec.Ports) > 0 {
		if existing.Spec.Ports[0].Port != desired.Spec.Ports[0].Port ||
			existing.Spec.Ports[0].TargetPort != desired.Spec.Ports[0].TargetPort ||
			existing.Spec.Ports[0].Protocol != desired.Spec.Ports[0].Protocol {
			return true
		}
	}

	return false
}

// updateStatus 更新 VibeApp 资源的状态（Status 子资源）
// 状态更新不会触发新的 reconcile 循环，这是写入 Status 的正确方式
func (r *VibeAppReconciler) updateStatus(ctx context.Context, vibesapp *appsexamplecomv1alpha1.VibeApp) {
	log := logf.FromContext(ctx).WithValues("vibeapp", vibesapp.Name, "namespace", vibesapp.Namespace)

	// 更新 ObservedGeneration：记录控制器最后一次处理的 Generation
	// Generation 在 Spec 变化时递增，用于检测 stale status 更新
	vibesapp.Status.ObservedGeneration = vibesapp.Generation

	// 设置 Available condition
	// Condition 类型遵循 Kubernetes 惯例：Available、Progressing、Degraded
	now := metav1.NewTime(time.Now())
	conditionType := "Available"
	conditionStatus := metav1.ConditionTrue
	conditionReason := "ReconciliationSuccess"
	conditionMessage := "VibeApp has been reconciled successfully"

	// 查找是否已存在同类型的 condition
	existingIdx := -1
	for i, cond := range vibesapp.Status.Conditions {
		if cond.Type == conditionType {
			existingIdx = i
			break
		}
	}

	newCondition := metav1.Condition{
		Type:               conditionType,
		Status:             conditionStatus,
		Reason:             conditionReason,
		Message:            conditionMessage,
		LastTransitionTime: now, // 记录状态 transition 时间
	}

	if existingIdx >= 0 {
		// 仅当状态实际变化时才更新，避免打刷大量状态更新
		existingCond := vibesapp.Status.Conditions[existingIdx]
		if existingCond.Status != newCondition.Status ||
			existingCond.Reason != newCondition.Reason ||
			existingCond.Message != newCondition.Message {
			vibesapp.Status.Conditions[existingIdx] = newCondition
			log.V(1).Info("Updating VibeApp status condition",
				"condition", conditionType,
				"status", conditionStatus,
				"reason", conditionReason)
		}
	} else {
		// 新增 condition
		vibesapp.Status.Conditions = append(vibesapp.Status.Conditions, newCondition)
		log.V(1).Info("Adding new VibeApp status condition",
			"condition", conditionType,
			"status", conditionStatus)
	}

	// 更新状态（使用 Status() 而不是 Update()，这是更新 status 子资源的正确方式）
	if err := r.Status().Update(ctx, vibesapp); err != nil {
		log.Error(err, "Failed to update VibeApp status")
		// 注意：状态更新失败不应该导致 reconcile 失败
		// 因为 Status 是可选信息，主资源的 Spec 已经同步完成
	}
}

// labelsForVibeApp 返回 VibeApp 的标准标签集合
// 这些标签用于：
//   - 标识资源由本 Operator 管理
//   - 作为 selector 选择相关的 Pods
//   - 便于 kubectl 查询和过滤
func (r *VibeAppReconciler) labelsForVibeApp(vibesapp *appsexamplecomv1alpha1.VibeApp) map[string]string {
	labels := map[string]string{
		"app.kubernetes.io/name":       "vibeapp",       // 应用名称
		"app.kubernetes.io/instance":   vibesapp.Name,   // 实例名（CR 名称）
		"app.kubernetes.io/managed-by": "vibe-operator", // 标识由本 Operator 管理
	}
	return labels
}

// SetupWithManager 配置控制器与 Manager 的集成
// 这是 controller-runtime 要求的接口实现
//
// 配置内容包括：
//   - For: 监听 VibeApp 资源的变化
//   - Owns: 声明 Deployment 和 Service 是 VibeApp 的子资源
//     当 VibeApp 被删除时，自动级联删除这些资源
//     当这些资源变化时，也触发 VibeApp 的 reconcile
func (r *VibeAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsexamplecomv1alpha1.VibeApp{}). // 主资源
		Owns(&appsv1.Deployment{}).             // 子资源：Deployment
		Owns(&corev1.Service{}).                // 子资源：Service
		Named("vibeapp").                       // 控制器名称
		Complete(r)                             // 完成配置
}
