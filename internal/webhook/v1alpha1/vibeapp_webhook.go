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

package v1alpha1

import (
	"context"
	"fmt"
	"reflect"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	appsexamplecomv1alpha1 "apps.example.com/vibe-operator/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var vibeapplog = logf.Log.WithName("vibeapp-resource")

// SetupVibeAppWebhookWithManager registers the webhook for VibeApp in the manager.
// 这是 manager 启动时调用的入口函数，用于注册本包中的 webhook 处理器
//
// 注册的 webhook 类型包括：
//   - Defaulting Webhook（变更类型）：在资源创建/更新前修改对象
//   - Validating Webhook（校验类型）：在资源持久化前验证对象
func SetupVibeAppWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &appsexamplecomv1alpha1.VibeApp{}).
		WithValidator(&VibeAppCustomValidator{}).
		WithDefaulter(&VibeAppCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-apps-example-com-v1alpha1-vibeapp,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.example.com,resources=vibeapps,verbs=create;update,versions=v1alpha1,name=mvibeapp-v1alpha1.kb.io,admissionReviewVersions=v1
//
// VibeAppCustomDefaulter 是变更 Webhook 处理器，负责为 VibeApp 资源设置默认值
// 它在以下时机被调用：
//   - 创建 VibeApp 资源时（create 操作）
//   - 更新 VibeApp 资源时（update 操作）
//
// 默认化（Mutating）Webhook 的特点：
//   - 可以修改传入的对象，修改会持久化到 etcd
//   - 按注册顺序执行（如果有多个）
//   - failurePolicy=fail 表示失败时拒绝整个请求
//
// 注意：+kubebuilder:object:generate=false 标记禁止生成 DeepCopy 方法
//
//	因为这个结构体仅用于 webhook 临时操作，不需要序列化或深拷贝
type VibeAppCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
	// 如果需要缓存或配置，可以在这里添加字段
}

// Default 实现 webhook.CustomDefaulter 接口
// 这是 admission webhook 的核心方法，API Server 在对象持久化前会调用它
//
// 参数：
//   - ctx: 上下文，包含日志、取消信号等
//   - obj: 待默认化的 VibeApp 对象（可修改）
//
// 返回值：
//   - error: 如果返回错误，整个创建/更新操作会被拒绝
func (d *VibeAppCustomDefaulter) Default(ctx context.Context, obj *appsexamplecomv1alpha1.VibeApp) error {
	log := vibeapplog.WithValues("vibeapp", obj.GetName(), "namespace", obj.GetNamespace())
	log.Info("Applying default values")

	// 记录原始值以便调试和验证变更
	originalReplicas := obj.Spec.Replicas
	originalLabels := obj.Labels

	// === 默认化逻辑开始 ===

	// 1. 添加或覆盖 created-by 标签
	// 确保所有由本 Operator 管理的资源都能被识别
	if obj.Labels == nil {
		obj.Labels = map[string]string{}
	}
	obj.Labels["created-by"] = "wang" // 固定标识符，表明资源由本 Operator 创建

	// 2. 确保 replicas 字段存在且不小于 2
	// 需求：即使未指定或指定为 <2，也强制设置为 2
	// 这既是默认值设置，也是安全保护（防止用户忘记设置或设为 1）
	replicas := obj.Spec.Replicas
	if replicas < 2 {
		if replicas != 0 { // 0 通常表示未设置（int32 zero value）
			log.Info("Adjusting replicas to minimum",
				"original", originalReplicas,
				"adjusted", 2)
		}
		obj.Spec.Replicas = 2
	}

	// === 默认化逻辑结束 ===

	// 记录实际应用的变更（仅当确实有变更时）
	// 使用 reflect.DeepEqual 对比 map 内容
	if !reflect.DeepEqual(originalLabels, obj.Labels) || originalReplicas != obj.Spec.Replicas {
		log.Info("Defaults applied",
			"labels.created-by", obj.Labels["created-by"],
			"replicas", obj.Spec.Replicas)
	}

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-apps-example-com-v1alpha1-vibeapp,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.example.com,resources=vibeapps,verbs=create;update,versions=v1alpha1,name=vvibeapp-v1alpha1.kb.io,admissionReviewVersions=v1
//
// VibeAppCustomValidator 是校验 Webhook 处理器，负责验证 VibeApp 资源的正确性
// 它在对象持久化到 etcd 之前被调用，用于确保 Spec 符合业务规则
//
// 校验 Webhook 的特点：
//   - 只读，不能修改对象
//   - 失败时返回错误，请求被拒绝
//   - 失败策略由 MutatingWebhookConfiguration.failurePolicy 控制
type VibeAppCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
	// 例如：可以缓存正则表达式、配置参数等
}

// ValidateCreate 实现 webhook.CustomValidator 接口，处理创建操作的验证
// 该方法在以下时机被调用：
//   - 用户执行 `kubectl apply` 或 `kubectl create` 创建新的 VibeApp
//   - 控制器或其他组件通过 API 创建 VibeApp
//
// 参数：
//   - ctx: 上下文对象
//   - obj: 待验证的 VibeApp 对象（只读）
//
// 返回值：
//   - admission.Warnings: 可选的警告信息列表（非阻塞）
//   - error: 验证失败时返回错误，成功返回 nil
func (v *VibeAppCustomValidator) ValidateCreate(_ context.Context, obj *appsexamplecomv1alpha1.VibeApp) (admission.Warnings, error) {
	log := vibeapplog.WithValues("vibeapp", obj.GetName(), "namespace", obj.GetNamespace())
	log.Info("Validating VibeApp creation")

	// === 验证规则 ===
	// 所有验证规则都是"必填"（required）或"格式"（format）校验
	// 注意：副本数的最小值检查同时由 mutating webhook 和 controller 保证
	//       这里做双重验证，提供更即时的错误反馈给用户

	// 1. 验证 image 字段必填
	// image 指定容器镜像，是运行应用必需的
	if obj.Spec.Image == "" {
		err := fmt.Errorf("spec.image is required")
		log.Error(err, "Validation failed: image is required")
		return nil, err
	}

	// 2. 验证 port 字段必填且大于 0
	// port 指定容器监听端口，必须是有效的 TCP 端口号（1-65535）
	if obj.Spec.Port <= 0 {
		err := fmt.Errorf("spec.port must be a positive integer, got %d", obj.Spec.Port)
		log.Error(err, "Validation failed: invalid port")
		return nil, err
	}

	// 3. 验证 replicas 字段必填且不小于 2
	// replicas 指定期望副本数，最小值为 2（HA 要求）
	// 如果用户设置 <2，mutating webhook 会将其改为 2，但这里双重检查
	if obj.Spec.Replicas < 2 {
		err := fmt.Errorf("spec.replicas must be at least 2, got %d", obj.Spec.Replicas)
		log.Error(err, "Validation failed: replicas too low")
		return nil, err
	}

	// 4. 验证 healthCheckPath 必填
	// healthCheckPath 用于配置 Pod 的 liveness/readiness probes
	if obj.Spec.HealthCheckPath == "" {
		err := fmt.Errorf("spec.healthCheckPath is required")
		log.Error(err, "Validation failed: healthCheckPath missing")
		return nil, err
	}

	// 5. 验证 storagePath 必填
	// storagePath 指定宿主机路径，用于 HostPath 卷挂载
	if obj.Spec.StoragePath == "" {
		err := fmt.Errorf("spec.storagePath is required")
		log.Error(err, "Validation failed: storagePath missing")
		return nil, err
	}

	// 所有验证通过
	log.Info("Validation passed")
	return nil, nil
}

// ValidateUpdate 实现 webhook.CustomValidator 接口，处理更新操作的验证
// 该方法在以下时机被调用：
//   - 用户执行 `kubectl apply` 更新已有 VibeApp
//   - 控制器或其他组件通过 API 更新 VibeApp
//
// 参数：
//   - ctx: 上下文对象
//   - oldObj: 更新前的 VibeApp 对象（用于对比变化）
//   - newObj: 更新后的 VibeApp 对象（待验证）
//
// 返回值：
//   - admission.Warnings: 可选的警告信息列表
//   - error: 验证失败时返回错误
func (v *VibeAppCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj *appsexamplecomv1alpha1.VibeApp) (admission.Warnings, error) {
	log := vibeapplog.WithValues("vibeapp", newObj.GetName(), "namespace", newObj.GetNamespace())
	log.Info("Validating VibeApp update")

	// 更新验证逻辑与创建验证相同
	// 注意：我们只验证 newObj（新状态），不关心从 oldObj 变更为 newObj 的过程
	// 如果需要验证特定字段是否允许修改，可以在这里添加对比 oldObj 的逻辑

	// 1. 验证 image 字段必填
	if newObj.Spec.Image == "" {
		err := fmt.Errorf("spec.image is required")
		log.Error(err, "Validation failed: image is required")
		return nil, err
	}

	// 2. 验证 port 字段必填且大于 0
	if newObj.Spec.Port <= 0 {
		err := fmt.Errorf("spec.port must be a positive integer, got %d", newObj.Spec.Port)
		log.Error(err, "Validation failed: invalid port")
		return nil, err
	}

	// 3. 验证 replicas 字段必填且不小于 2
	if newObj.Spec.Replicas < 2 {
		err := fmt.Errorf("spec.replicas must be at least 2, got %d", newObj.Spec.Replicas)
		log.Error(err, "Validation failed: replicas too low")
		return nil, err
	}

	// 4. 验证 healthCheckPath 必填
	if newObj.Spec.HealthCheckPath == "" {
		err := fmt.Errorf("spec.healthCheckPath is required")
		log.Error(err, "Validation failed: healthCheckPath missing")
		return nil, err
	}

	// 5. 验证 storagePath 必填
	if newObj.Spec.StoragePath == "" {
		err := fmt.Errorf("spec.storagePath is required")
		log.Error(err, "Validation failed: storagePath missing")
		return nil, err
	}

	// 验证通过
	log.Info("Validation passed")
	return nil, nil
}

// ValidateDelete 实现 webhook.CustomValidator 接口，处理删除操作的验证
// 该方法在删除 VibeApp 资源前被调用，可用于防止意外删除或清理关联资源
//
// 注意：当前实现为空，允许所有删除操作
// 如果需要阻止删除（例如生产环境保护），可以在此添加逻辑返回错误
func (v *VibeAppCustomValidator) ValidateDelete(_ context.Context, obj *appsexamplecomv1alpha1.VibeApp) (admission.Warnings, error) {
	vibeapplog.Info("Validation for VibeApp upon deletion", "name", obj.GetName())

	// TODO(user): fill in your validation logic upon object deletion.
	// 示例：阻止删除生产环境资源
	// if obj.Labels["environment"] == "production" {
	//     return nil, fmt.Errorf("deleting production VibeApp is not allowed")
	// }

	return nil, nil
}
