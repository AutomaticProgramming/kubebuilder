# VibeApp Operator

VibeApp Operator 是一个基于 Kubebuilder 框架构建的 Kubernetes Operator，用于管理 `VibeApp` 自定义资源。它通过将 VibeApp 规范自动转换为生产就绪的 Deployment 和 Service，实现应用工作负载的自动化部署。

## 功能特性

- **自定义资源定义 (CRD):** 使用简洁的字段定义应用：`image`、`replicas`、`healthCheckPath`、`storagePath`、`port`
- **变更 Webhook:** 自动添加 `created-by: "wang"` 标签，确保 `replicas` 默认为 2（最小 2）
- **校验 Webhook:** 强制验证必填字段（`image`、`port`、`replicas`、`healthCheckPath`、`storagePath`）
- **调和循环 (Reconciliation Loop):** 创建和管理 Deployment 与 Service，并设置 OwnerReference 实现级联删除
- **自愈能力:** 自动纠正手动缩容到 2 个副本以下的情况，保持高可用
- **TLS 安全:** 使用自签名证书保障 Webhook 通信安全

## 架构概览

```
┌─────────────────┐
│   VibeApp CR    │  (用户创建的资源)
│   apps.example.com/v1alpha1
└────────┬────────┘
         │ 1. 创建/更新/删除事件
         ▼
┌─────────────────────────────────────────────────────┐
│              VibeApp Operator                       │
│  ┌─────────────────────────────────────────────┐   │
│  │  MutatingWebhook (变更)                     │   │
│  │  - 添加 created-by 标签                     │   │
│  │  - 设置默认 replicas=2                      │   │
│  └─────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────┐   │
│  │  ValidatingWebhook (校验)                   │   │
│  │  - 验证必填字段                             │   │
│  │  - 检查 replicas ≥ 2                        │   │
│  └─────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────┐   │
│  │  Controller (调和循环)                      │   │
│  │  - 创建/更新 Deployment                     │   │
│  │    • HostPath 卷挂载到 /data                │   │
│  │    • HTTP 存活/就绪探针                    │   │
│  │  - 创建/更新 Service (ClusterIP)           │   │
│  │  - 自愈：修复副本数 < 2                    │   │
│  │  - 更新 Status.Conditions                  │   │
│  └─────────────────────────────────────────────┘   │
└─────────┬─────────────────────────────────────────┘
          │
    ┌─────┴─────┐
    ▼           ▼
┌──────────────────────────────────────┐
│  Deployment (vibeapp-sample)         │
│  • 副本数: 2                         │
│  • 镜像: nginx:latest                │
│  • 卷挂载: HostPath /data            │
│  • 探针: /healthz on port 80        │
└──────────────────────────────────────┘

┌──────────────────────────────────────┐
│  Service (vibeapp-sample)            │
│  • 类型: ClusterIP                   │
│  • 端口: 80 → 80 (targetPort)       │
│  • Selector: app.kubernetes.io/instance=vibeapp-sample
└──────────────────────────────────────┘
```

## 快速开始

### 前置要求

- **Go:** 1.20+ 版本
- **Kubebuilder:** v4.x (用于项目脚手架和代码生成)
- **Kubernetes 集群:** v1.24+ (本地可使用 Minikube、Kind 或 k3s)
- **kubectl:** 与集群版本匹配
- **Docker:** 用于构建和推送镜像
- **Git:** 用于版本控制

### 1. 克隆项目

```bash
git clone <your-repo>/vibe-operator.git
cd vibe-operator
```

### 2. 构建 Operator 镜像

```bash
# 设置镜像名称（替换为你的镜像仓库）
export IMG=your-registry/vibe-operator:latest

# 构建并推送镜像
make docker-build IMG=$IMG
make docker-push IMG=$IMG
```

### 3. 安装 CRD 到集群

```bash
make install
```

验证 CRD 是否安装成功：

```bash
kubectl get crd vibeapps.apps.example.com
```

### 4. 部署 Operator 到集群

```bash
make deploy IMG=$IMG
```

验证 Operator 是否正常运行：

```bash
kubectl get deployments -n kubebuilder-system
kubectl logs -n kubebuilder-system deployment/kubebuilder-vibeapp-operator-controller-manager
```

### 5. 创建 VibeApp 实例

使用示例配置文件：

```bash
kubectl apply -f config/samples/apps.example.com_v1alpha1_vibeapp.yaml
```

或者自定义创建：

```yaml
# my-vibeapp.yaml
apiVersion: apps.example.com/v1alpha1
kind: VibeApp
metadata:
  name: my-nginx
  namespace: default
spec:
  image: nginx:1.25
  replicas: 3
  healthCheckPath: /healthz
  storagePath: /var/lib/nginx/data
  port: 80
```

应用配置：

```bash
kubectl apply -f my-vibeapp.yaml
```

### 6. 验证资源

检查 VibeApp 状态：

```bash
kubectl get vibeapp my-nginx -o wide
kubectl describe vibeapp my-nginx
```

查看自动创建的 Deployment 和 Service：

```bash
kubectl get deployments -l app.kubernetes.io/instance=my-nginx
kubectl get services -l app.kubernetes.io/instance=my-nginx
```

查看 Pod 详情：

```bash
kubectl get pods -l app.kubernetes.io/instance=my-nginx
kubectl describe pod <pod-name>
```

## CRD 使用说明

### VibeApp 规范说明

| 字段 | 类型 | 必填 | 描述 | 示例值 |
|------|------|------|------|--------|
| `image` | string | 是 | 容器镜像地址 | `nginx:latest` |
| `replicas` | int32 | 是 | 期望副本数（最小 2） | `2` |
| `healthCheckPath` | string | 是 | HTTP 健康检查路径 | `/healthz` |
| `storagePath` | string | 是 | 宿主机挂载路径（HostPath） | `/data` |
| `port` | int32 | 是 | 容器和服务端口 | `80` |

**重要约束：**

- `replicas` 最小值强制为 2，即使设置为 0 或 1，也会自动调整为 2
- `port` 必须是正整数（1-65535）
- `storagePath` 对应 Kubernetes 节点的实际路径，确保节点上有该目录

### 示例场景

#### 场景 1: 部署一个简单的 Web 应用

```yaml
apiVersion: apps.example.com/v1alpha1
kind: VibeApp
metadata:
  name: webapp
spec:
  image: nginx:alpine
  replicas: 2
  healthCheckPath: /
  storagePath: /tmp/data
  port: 80
```

#### 场景 2: 更新应用镜像版本

```bash
# 编辑 CR
kubectl edit vibeapp webapp

# 修改 spec.image 为新版本
# spec:
#   image: nginx:1.25.3
```

Operator 会检测到变化并触发滚动更新。

#### 场景 3: 测试自愈能力

```bash
# 查看当前副本数
kubectl get deployment webapp

# 手动缩容到 1（模拟异常）
kubectl scale deployment webapp --replicas=1

# 等待几秒，Operator 会自动恢复到 2 或指定值
kubectl get deployment webapp
```

### 查看状态条件（Conditions）

VibeApp 状态中包含 `Conditions` 数组，用于表示资源的整体状态：

```bash
kubectl get vibeapp webapp -o jsonpath='{.status.conditions}'
```

可能的条件类型：

- `Available`: 表示 VibeApp 已成功调和（状态为 True）
- `Degraded`: 表示调和过程中出现错误（状态为 True）
- `Progressing`: 表示正在创建或更新中

## 开发与测试

### 本地开发

运行本地 Operator（连接到当前 kubeconfig 集群）：

```bash
make run
```

或者使用 Docker 运行：

```bash
make run USE_DOCKER=1
```

### 运行测试

**单元测试：**

```bash
make test
```

**集成测试（需要集群）：**

```bash
# 1. 安装 CRD
make install

# 2. 运行 operator（本地或部署）
make run

# 3. 在另一个终端创建测试 CR
kubectl apply -f config/samples/apps.example.com_v1alpha1_vibeapp.yaml

# 4. 验证资源是否正确创建
kubectl get vibeapps,deployments,services
```

### 代码生成

当修改了 API 类型定义后，需要重新生成代码：

```bash
# 生成 DeepCopy 等方法
make generate

# 生成 CRD 和 Webhook manifests
make manifests
```

### 更新 CRD 示例文件

编辑 `config/samples/apps.example.com_v1alpha1_vibeapp.yaml`，然后重新生成：

```bash
make generate manifests
```

## Webhook 机制

### MutatingWebhook（变更）

**触发时机：** 创建或更新 VibeApp 时

**执行操作：**
1. 添加或覆盖 `metadata.labels["created-by"] = "wang"`
2. 确保 `spec.replicas` 不小于 2

**示例：**
```yaml
# 用户提交的 CR（replicas: 1）
apiVersion: apps.example.com/v1alpha1
kind: VibeApp
metadata:
  name: test
spec:
  image: nginx:latest
  replicas: 1  # ← 会被自动修改为 2
  healthCheckPath: /healthz
  storagePath: /data
  port: 80
```

Webhook 修改后的实际存储：
- `replicas` = 2

### ValidatingWebhook（校验）

**触发时机：** 创建或更新 VibeApp 时（在 MutatingWebhook 之后）

**验证规则：**
- `spec.image` 必须非空
- `spec.port` 必须大于 0
- `spec.replicas` 必须 ≥ 2
- `spec.healthCheckPath` 必须非空
- `spec.storagePath` 必须非空

**失败示例：**
```bash
# 尝试创建缺少 image 的 CR
kubectl apply -f - <<EOF
apiVersion: apps.example.com/v1alpha1
kind: VibeApp
metadata:
  name: invalid
spec:
  replicas: 2
  healthCheckPath: /healthz
  storagePath: /data
  port: 80
EOF
```
输出：
```
Error from server (BadRequest): error when creating "STDIN":
admission webhook "validation.apps.example.com/v1alpha1" denied the request: spec.image is required
```

### TLS 证书

Operator 使用自签名证书保护 Webhook 通信。证书生成流程：

1. CA 证书和私钥：`certs/ca.crt`、`certs/ca.key`
2. Webhook 服务器证书：`certs/tls.crt`、`certs/tls.key`
3. 部署时通过 `secretGenerator` 创建 `kubebuilder-webhook-server-cert` secret
4. CA Bundle 自动注入到 `MutatingWebhookConfiguration` 和 `ValidatingWebhookConfiguration`

如果需要重新生成证书：

```bash
# 删除旧证书
rm -rf certs/*

# 生成新证书（CN 必须与 webhook service 匹配）
cd certs
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -subj "/CN=ca.example.com" -days 3650 -out ca.crt

openssl genrsa -out tls.key 2048
openssl req -new -key tls.key -subj "/CN=webhook-service.kubebuilder-system.svc" -out server.csr
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out tls.crt -days 365

# 重新部署
cd ..
make deploy IMG=$IMG
```

## 故障排除

### 1. Operator 无法启动

**症状：** Pod 处于 `CrashLoopBackOff` 或 `Error` 状态

**排查步骤：**

```bash
# 查看 Pod 日志
kubectl logs -n kubebuilder-system deployment/kubebuilder-vibeapp-operator-controller-manager

# 查看 Pod 详情
kubectl describe pod -n kubebuilder-system <pod-name>
```

**常见原因：**
- 镜像拉取失败：检查 `IMAGE` 参数是否正确，是否有权限访问仓库
- 配置错误：检查 `config/default/kustomization.yaml` 配置
- 证书问题：确保 `certs/` 目录下有有效的 TLS 证书

### 2. CRD 无法创建

**症状：** `kubectl apply -f my-vibeapp.yaml` 报错 "the server does not have a resource type"

**解决：**

```bash
# 确认 CRD 已安装
kubectl get crd vibeapps.apps.example.com

# 如果未安装，重新安装
make install
```

### 3. Webhook 拒绝合法请求

**症状：** 创建 VibeApp 时被 webhook 拒绝，但所有字段都正确

**排查：**

```bash
# 查看 webhook 配置
kubectl get mutatingwebhookconfiguration mvibeapp-v1alpha1.kb.io -o yaml
kubectl get validatingwebhookconfiguration vvibeapp-v1alpha1.kb.io -o yaml

# 检查 webhook service 和证书
kubectl get service -n kubebuilder-system webhook-service
kubectl get secret -n kubebuilder-system kubebuilder-webhook-server-cert

# 查看 Operator 日志中的 webhook 错误
kubectl logs -n kubebuilder-system deployment/kubebuilder-vibeapp-operator-controller-manager | grep -i webhook
```

**常见原因：**
- CA Bundle 未正确注入：检查 `config/default/webhook_ca_patch.yaml` 是否正确应用
- Service 命名空间不匹配：webhook configuration 中的 `namespace` 应为 `kubebuilder-system`
- 证书过期：重新生成证书并重新部署

### 4. Deployment 或 Service 未创建

**排查步骤：**

```bash
# 检查 VibeApp 状态
kubectl get vibeapp <name> -o yaml

# 查看 Operator 日志
kubectl logs -n kubebuilder-system deployment/kubebuilder-vibeapp-operator-controller-manager

# 确认 OwnerReference 设置正确
kubectl get deployment <name> -o yaml | grep ownerReferences -A5
```

**常见原因：**
- RBAC 权限不足：检查 `config/rbac/role.yaml` 是否包含对 deployments 和 services 的 CRUD 权限
- Spec 字段有问题：确保所有必填字段都已设置
- 调和错误：查看 Operator 日志了解具体错误信息

### 5. 自愈功能不工作

**症状：** 手动缩放 Deployment 副本数后，Operator 未自动恢复

**排查：**

```bash
# 1. 确认手动缩放确实改变了副本数
kubectl get deployment <name>

# 2. 等待至少一个 reconcile 周期（默认 10 小时？实际上会立即触发）
#    可以修改 VibeApp 的 annotation 强制触发：
kubectl annotate vibeapp <name> vibeapp.example.com/lastReconcile="$(date +%s)"

# 3. 查看 Operator 日志，搜索 "Fixing Deployment replicas"
kubectl logs -n kubebuilder-system deployment/kubebuilder-vibeapp-operator-controller-manager | grep -i "self-healing"
```

**自愈逻辑说明：**
- Operator 会在每次 reconcile 时检查owned Deployment 的副本数
- 如果副本数 < `max(VibeApp.Spec.Replicas, 2)`，会通过 Patch 修复
- 修复是幂等的，多次触发不会造成问题

### 6. 证书过期

Webhook 证书默认有效期为 365 天。过期后 API Server 会拒绝调用 Webhook。

**检查证书过期时间：**

```bash
kubectl get secret -n kubebuilder-system kubebuilder-webhook-server-cert -o jsonpath='{.data.tls.crt}' | base64 -d | openssl x509 -noout -dates
```

**更新证书：** 删除 `certs/` 目录下的文件，重新运行上面的证书生成步骤，然后重新部署 Operator。

## 高级配置

### 调整 Webhook 失败策略

默认 `failurePolicy: Fail` 表示 webhook 失败时拒绝请求。可以修改为 `Ignore` 让请求继续（不推荐生产环境）。

编辑 `config/webhook/manifests.yaml`：

```yaml
webhooks:
- name: mvibeapp-v1alpha1.kb.io
  failurePolicy: Fail  # 改为 Ignore 会在 webhook 失败时跳过
```

### 启用 Leader Election

如果要在多副本部署时避免脑裂，启用 leader election：

```bash
kubectl edit deployment -n kubebuilder-system kubebuilder-vibeapp-operator-controller-manager
```

添加环境变量：

```yaml
spec:
  template:
    spec:
      containers:
      - name: manager
        env:
        - name: ENABLE_LEADER_ELECTION
          value: "true"
```

或者在命令行参数中添加 `--leader-elect=true`（默认已启用）。

### 自定义资源配额和限制

编辑 `config/default/manager.yaml`（kustomize 会在部署时应用）或直接修改部署：

```bash
kubectl edit deployment -n kubebuilder-system kubebuilder-vibeapp-operator-controller-manager
```

添加资源限制：

```yaml
resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 256Mi
```

### 调整 Reconcile 并发数

默认情况下，controller-runtime 会并发调和多个资源。可以通过设置 `--max-concurrent-reconciles` 调整：

```bash
# 编辑 deployment
kubectl edit deployment -n kubebuilder-system kubebuilder-vibeapp-operator-controller-manager
```

在 `spec.template.spec.containers[0].command` 中添加：

```yaml
- --max-concurrent-reconciles=5
```

## 贡献指南

欢迎贡献代码、报告 Issue 或提出建议！

### 开发工作流

1. Fork 本项目
2. 创建特性分支: `git checkout -b feature/AmazingFeature`
3. 提交更改: `git commit -m 'Add some AmazingFeature'`
4. 推送到分支: `git push origin feature/ AmazingFeature`
5. 创建 Pull Request

### 代码规范

- 遵循 Go 代码风格（使用 `go fmt` 和 `go vet`）
- 为新功能添加单元测试
- 为新函数/结构体添加注释（支持 godoc）
- 确保 `make test` 通过

### 提交信息格式

推荐使用 Conventional Commits 格式：

```
feat: add support for configmap volume mounts
fix: handle nil pointer in reconcileDeployment
docs: update README with deployment examples
test: add unit tests for VibeAppCustomValidator
```

## 更多资源

- [Kubebuilder 官方文档](https://book.kubebuilder.io/introduction.html)
- [Kubernetes Operator 最佳实践](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [controller-runtime API 参考](https://pkg.go.dev/sigs.k8s.io/controller-runtime)

## License

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
