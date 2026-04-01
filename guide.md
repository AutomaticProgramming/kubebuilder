# Kubernetes Operator 开发指南：极简应用编排器

## 1. 项目愿景
开发一个基于 **Kubebuilder** 框架的 Operator，用于管理自定义资源（CR）。该 Operator 能够自动将用户的配置转化为生产就绪的 `Deployment` 和 `Service`，并在安全合规、配置校验和状态自愈方面提供自动化支持。

## 2. API 定义 (CRD)
**资源组 (Group):** `apps.example.com`
**版本 (Version):** `v1alpha1`
**种类 (Kind):** `VibeApp`

### Spec 字段要求：
* **image (string):** 容器镜像地址。
* **replicas (int32):** 期望副本数。
* **healthCheckPath (string):** HTTP 健康检查的路径（如 `/healthz`）。
* **storagePath (string):** 需要挂载的宿主机路径（HostPath）。
* **port (int32):** 容器监听及服务暴露的端口号。

---

## 3. 准入控制 (Webhooks)

### 3.1 变更插件 (Mutating Webhook)
* **逻辑：** 在对象持久化到 ETCD 之前，拦截请求。
* **动作：** 检查 `metadata.labels`。如果不存在 `created-by` 标签，则自动添加 `created-by: "wang"`。如果已存在，则覆盖为该值。检查'replicas'字段是否存在，不存在自动添加该字段，大小为2，如果存在但值小于2，自动将值修改为2

### 3.2 校验插件 (Validating Webhook)
* **必填项校验：** 确保 `image`、`port`、`replicas` 均不为空且合法。
* **逻辑拦截：** 如果上述字段缺失，API Server 应拒绝该创建/更新请求并返回错误信息。

---

## 4. 控制循环逻辑 (Reconciliation Loop)



### 4.1 资源映射规则
当 `VibeApp` 资源被创建时，Operator 需同步创建以下底层资源：
1.  **Deployment:**
    * **存储挂载：** 创建一个 `volume`，类型为 `HostPath`，指向 `spec.storagePath`。将其挂载到容器内的 `/data` 目录。
    * **存活探针 (Liveness Probe):** 使用 `HTTPGet` 方式，路径为 `spec.healthCheckPath`，端口为 `spec.port`。
2.  **Service:**
    * **类型：** `ClusterIP`。
    * **端口：** 将 Service 端口与容器端口（`spec.port`）对齐。

### 4.2 级联删除与所有权
* 使用 `controller-runtime` 的 `SetControllerReference` 方法，将 `VibeApp` 设置为 Deployment 和 Service 的 **Owner**，确保 CR 被删除时，关联资源自动清理。

### 4.3 状态漂移修复 (Self-healing)
* **监控副本数：** 如果用户手动通过 `kubectl scale` 更改了底层 Deployment 的副本数（使其小于 2），Operator 在下一轮调谐中必须将其强制改回 `2`（或用户定义的更大值）。

---

## 5. 网络安全与证书配置 (TLS)

### 5.1 单向认证配置
* **Operator 端：** 作为一个 HTTPS 服务，Operator 的 Webhook Server 必须配置证书。
* **证书策略：** 采用**自签名证书 (Self-signed Certificate)**。
* **APIServer 通讯：** 配置 `ValidatingWebhookConfiguration` 和 `MutatingWebhookConfiguration` 时，将 `caBundle` 设置为 Operator 自签证书的根证书，使 APIServer 能够信任 Operator。
* **客户端校验：** 明确要求 Operator 在调用 APIServer 时跳过证书校验（单向认证场景），这通常通过配置 `rest.Config` 中的 `Insecure = true` 或忽略 CA 验证来实现。

---

## 6. 开发环境与工具栈建议
* **框架:** Kubebuilder v3+
* **语言:** Go 1.20+
* **测试工具:** EnvTest (本地测试) 或 Makefile 中预设的 `make install` & `make run` 流程。
* **部署:** 提供自定义的 `Kustomize` 配置，用于在集群内快速部署 Webhook 所需的证书 Secret。

