# VibeApp Operator 项目结构详解

```
vibe-operator/
├── AGENTS.md
├── api
│   └── v1alpha1
│       ├── groupversion_info.go
│       ├── vibeapp_types.go
│       └── zz_generated.deepcopy.go
├── bin
│   ├── controller-gen -> /Users/suhuanzhen/Documents/kubebuilder/bin/controller-gen-v0.20.1
│   ├── controller-gen-v0.20.1
│   ├── k8s
│   ├── manager
│   ├── setup-envtest -> /Users/suhuanzhen/Documents/kubebuilder/bin/setup-envtest-release-0.23
│   ├── setup-envtest-release-0.23
│   └── vibe-operator
├── CLAUDE.md
├── cmd
│   └── main.go
├── config
│   ├── certmanager
│   │   ├── certificate-metrics.yaml
│   │   ├── certificate-webhook.yaml
│   │   ├── issuer.yaml
│   │   ├── kustomization.yaml
│   │   └── kustomizeconfig.yaml
│   ├── crd
│   │   ├── bases
│   │   │   └── apps.example.com_vibeapps.yaml
│   │   ├── kustomization.yaml
│   │   └── kustomizeconfig.yaml
│   ├── default
│   │   ├── cert_metrics_manager_patch.yaml
│   │   ├── certs
│   │   │   ├── ca.crt
│   │   │   ├── ca.key
│   │   │   ├── ca.srl
│   │   │   ├── server.csr
│   │   │   ├── tls.crt
│   │   │   └── tls.key
│   │   ├── kustomization.yaml
│   │   ├── manager_metrics_patch.yaml
│   │   ├── manager_webhook_patch.yaml
│   │   ├── metrics_service.yaml
│   │   └── webhook_ca_patch.yaml
│   ├── manager
│   │   ├── kustomization.yaml
│   │   └── manager.yaml
│   ├── network-policy
│   │   ├── allow-metrics-traffic.yaml
│   │   ├── allow-webhook-traffic.yaml
│   │   └── kustomization.yaml
│   ├── prometheus
│   │   ├── kustomization.yaml
│   │   ├── monitor_tls_patch.yaml
│   │   └── monitor.yaml
│   ├── rbac
│   │   ├── kustomization.yaml
│   │   ├── leader_election_role_binding.yaml
│   │   ├── leader_election_role.yaml
│   │   ├── metrics_auth_role_binding.yaml
│   │   ├── metrics_auth_role.yaml
│   │   ├── metrics_reader_role.yaml
│   │   ├── role_binding.yaml
│   │   ├── role.yaml
│   │   ├── service_account.yaml
│   │   ├── vibeapp_admin_role.yaml
│   │   ├── vibeapp_editor_role.yaml
│   │   └── vibeapp_viewer_role.yaml
│   ├── samples
│   │   ├── apps.example.com_v1alpha1_vibeapp.yaml
│   │   └── kustomization.yaml
│   └── webhook
│       ├── kustomization.yaml
│       ├── manifests.yaml
│       └── service.yaml
├── development-plan.md
├── Dockerfile
├── go.mod
├── go.sum
├── guide.md
├── hack
│   └── boilerplate.go.txt
├── internal
│   ├── controller
│   │   ├── suite_test.go
│   │   ├── vibeapp_controller_test.go
│   │   └── vibeapp_controller.go
│   └── webhook
│       └── v1alpha1
│           ├── vibeapp_webhook_test.go
│           ├── vibeapp_webhook.go
│           └── webhook_suite_test.go
├── Makefile
├── PROJECT
├── README.md
└── test
    ├── e2e
    │   ├── e2e_suite_test.go
    │   └── e2e_test.go
    └── utils
        └── utils.go
```

## 目录结构详细说明

### 📁 根目录文件

| 文件 | 说明 |
|------|------|
| `AGENTS.md` | Kubebuilder 自动生成的代理文档 |
| `CLAUDE.md` | Claude Code 工作指南（本项目的 AI 助手配置文件） |
| `development-plan.md` | 详细的开发计划文档 |
| `Dockerfile` | Operator 容器镜像构建文件 |
| `go.mod` | Go 模块依赖定义 |
| `go.sum` | Go 模块依赖校验和 |
| `guide.md` | 项目设计规范（原始需求文档） |
| `Makefile` | 项目自动化构建、测试、部署脚本 |
| `PROJECT` | Kubebuilder 项目元数据（version、domain、resources 等）|
| `README.md` | 项目英文说明文档 |

---

### 📂 `api/` - API 定义

Kubernetes 自定义资源（CRD）的 Go 类型定义。

```
api/v1alpha1/
├── groupversion_info.go    # API Group/Version 注册信息
├── vibeapp_types.go        # VibeApp CRD 的 Spec 和 Status 定义
└── zz_generated.deepcopy.go # 自动生成的 DeepCopy 方法（不要手动修改）
```

**关键文件：**
- `vibeapp_types.go`: 定义 `VibeApp` 结构体，包含 `Spec`（image、replicas、healthCheckPath、storagePath、port）和 `Status`（conditions、observedGeneration）
- `groupversion_info.go`: 注册 API Scheme，声明 Group 为 `apps.example.com`

---

### 📂 `cmd/` - 程序入口

Operator 的主入口点。

```
cmd/
└── main.go
```

**main.go 职责：**
- 初始化 Scheme（注册 API 类型）
- 创建 Manager（设置客户端、schema、metrics、webhook server）
- 注册 Reconciler 和 Webhook
- 启动 Manager（开始 reconciliation 循环）

---

### 📂 `config/` - Kubernetes 部署配置

使用 Kustomize 组织的部署清单。

#### `config/crd/` - CRD 清单
```
config/crd/
├── bases/
│   └── apps.example.com_vibeapps.yaml  # 自动生成的 CRD YAML
├── kustomization.yaml
└── kustomizeconfig.yaml
```

- `bases/`: 包含 CRD 的原始 YAML 定义
- 通过 `make install` 部署到集群

#### `config/default/` - 默认部署配置（生产/开发）

完整的 Operator 部署所需的所有资源。

```
config/default/
├── certs/                          # TLS 证书文件（自签名）
│   ├── ca.crt                     # CA 证书（用于 API Server 信任 webhook）
│   ├── tls.crt                    # Webhook 服务器证书
│   └── tls.key                    # Webhook 服务器私钥
├── kustomization.yaml             # Kustomize 入口配置
├── manager_webhook_patch.yaml     # 为 manager deployment 添加 webhook 证书挂载
├── manager_metrics_patch.yaml     # Metrics 相关补丁（可选）
├── metrics_service.yaml           # Metrics Service（可选）
└── webhook_ca_patch.yaml          # 将 CA bundle 注入到 webhook configurations
```

**`kustomization.yaml` 核心配置：**
- `namespace: kubebuilder-system`：部署的命名空间
- `namePrefix: kubebuilder-`：资源名前缀
- `resources`: 包含 crd、rbac、manager、webhook
- `secretGenerator`: 生成 `kubebuilder-webhook-server-cert` secret（从 certs/ 读取）
- `patches`: 应用 `manager_webhook_patch.yaml` 和 `webhook_ca_patch.yaml`

#### `config/manager/` - Manager Deployment

```
config/manager/
├── kustomization.yaml
└── manager.yaml
```

- `manager.yaml`: Operator 主容器的 Deployment 定义（镜像、端口、命令参数等）

#### `config/rbac/` - RBAC 权限

```
config/rbac/
├── kustomization.yaml
├── role.yaml                        # 主角色（VibeApp、Deployment、Service CRUD）
├── role_binding.yaml                # 角色绑定到 ServiceAccount
├── service_account.yaml             # ServiceAccount 定义
├── leader_election_role.yaml        # Leader election 所需角色
├── leader_election_role_binding.yaml
├── metrics_reader_role.yaml         # Metrics 读取权限
├── metrics_auth_role.yaml           # Metrics 认证权限
├── metrics_auth_role_binding.yaml   # Metrics 认证角色绑定
├── vibeapp_admin_role.yaml          # VibeApp 管理员角色（全权）
├── vibeapp_editor_role.yaml         # VibeApp 编辑角色（create/update）
└── vibeapp_viewer_role.yaml         # VibeApp 查看角色（get/list/watch）
```

#### `config/webhook/` - Webhook 配置

```
config/webhook/
├── kustomization.yaml
├── manifests.yaml                   # MutatingWebhookConfiguration & ValidatingWebhookConfiguration
└── service.yaml                     # Webhook Service（ClusterIP:443 -> manager pod:9443）
```

- `manifests.yaml`: 定义两个 webhook 配置，path 分别为 `/mutate-apps-example-com-v1alpha1-vibeapp` 和 `/validate-apps-example-com-v1alpha1-vibeapp`
- `service.yaml`: webhook service，供 API Server 调用

#### `config/certmanager/` - Cert-manager 集成（可选/已禁用）

```
config/certmanager/
├── certificate-webhook.yaml         # Certificate CR（用于 webhook 证书）
├── certificate-metrics.yaml         # Certificate CR（用于 metrics 证书）
├── issuer.yaml                      # SelfSigned Issuer
├── kustomization.yaml
└── kustomizeconfig.yaml
```

**注意：** 当前已禁用（resources 和 replacements 均已注释）。如需启用，需要取消注释并注释掉 manual secretGenerator。

#### `config/samples/` - CR 示例

```
config/samples/
├── apps.example.com_v1alpha1_vibeapp.yaml  # VibeApp 示例
└── kustomization.yaml
```

示例文件包含完整的 VibeApp 配置，可直接使用 `kubectl apply -f config/samples/` 创建测试 CR。

---

### 📂 `internal/` - 业务代码

不导出的内部包，包含控制器和 webhook 实现。

#### `internal/controller/` - 控制器

```
internal/controller/
├── vibeapp_controller.go      # VibeAppReconciler 实现
├── vibeapp_controller_test.go # 单元测试（Ginkgo）
└── suite_test.go              # 测试套件（EnvTest 环境初始化）
```

**`vibeapp_controller.go` 核心功能：**
- `Reconcile()`: 主调和循环
- `deploymentForVibeApp()`: 构建期望的 Deployment
- `reconcileDeployment()`: 调和 Deployment（创建、自愈、更新）
- `serviceForVibeApp()`: 构建期望的 Service
- `reconcileService()`: 调和 Service
- `updateStatus()`: 更新 VibeApp 状态
- `labelsForVibeApp()`: 生成标准标签
- `SetupWithManager()`: 注册 watches

#### `internal/webhook/v1alpha1/` - Webhook

```
internal/webhook/v1alpha1/
├── vibeapp_webhook.go          # Webhook 处理器（Default + Validate）
├── vibeapp_webhook_test.go     # Webhook 单元测试
└── webhook_suite_test.go       # 测试套件
```

**`vibeapp_webhook.go` 核心：**
- `VibeAppCustomDefaulter`: MutatingWebhook，添加标签、设置副本数默认值
- `VibeAppCustomValidator`: ValidatingWebhook，验证必填字段
- `SetupVibeAppWebhookWithManager()`: 注册 webhook 到 manager

---

### 📂 `hack/` - 辅助脚本

```
hack/
└── boilerplate.go.txt  # 代码生成时的许可证头部模板
```

---

### 📂 `test/` - 测试

```
test/
├── e2e/
│   ├── e2e_suite_test.go  # E2E 测试套件
│   └── e2e_test.go        # E2E 测试用例（集成测试）
└── utils/
    └── utils.go           # 测试工具函数
```

---

### 📂 其他隐藏目录

| 目录 | 说明 |
|------|------|
| `.claude/` | Claude Code 记忆和计划文件存储目录 |
| `.devcontainer/` | VS Code Dev Container 配置（开发环境）|
| `.github/` | GitHub Actions CI/CD 工作流配置 |
| `.github/workflows/` | CI 流水线定义（如 `ci.yaml`）|
| `bin/` | 第三方工具二进制文件（controller-gen、setup-envtest 等）|
| `.dockerignore` | Docker 构建忽略文件 |
| `.gitignore` | Git 忽略文件 |
| `.golangci.yml` | golangci-lint 配置 |

---

## 关键配置文件说明

### `PROJECT` - Kubebuilder 项目配置

```yaml
cliVersion: 4.13.1
domain: example.com
repo: apps.example.com/vibe-operator
resources:
  - api:
      crdVersion: v1
      controller: true
      domain: example.com
      group: apps.example.com
      kind: VibeApp
      path: apps.example.com/vibe-operator/api/v1alpha1
      version: v1alpha1
      webhooks:
        defaulting: true
        validation: true
```

### `Makefile` - 主要 Targets

| Target | 说明 |
|--------|------|
| `make generate` | 生成 DeepCopy、Conversion 代码 |
| `make manifests` | 生成 CRD、RBAC、Webhook manifests |
| `make build` | 编译 operator 二进制文件 |
| `make run` | 本地运行 operator（连接 kubeconfig 集群）|
| `make test` | 运行单元测试 |
| `make install` | 安装 CRD 到集群 |
| `make deploy` | 部署 operator 到集群（使用 kustomize）|
| `make docker-build` | 构建 Docker 镜像 |
| `make docker-push` | 推送 Docker 镜像到仓库 |
| `make undeploy` | 从集群删除 operator |
| `make uninstall` | 从集群删除 CRD |

---

## 数据流概览

```
用户创建 VibeApp CR
         ↓
API Server 持久化到 etcd
         ↓
MutatingWebhook（修改 spec）
  - 添加 created-by=wang
  - 确保 replicas ≥ 2
         ↓
ValidatingWebhook（验证）
  - 检查必填字段
  - 返回错误或通过
         ↓
API Server 保存对象
         ↓
VibeApp CR 事件触发 → Controller Reconcile
         ↓
Controller 创建/更新:
  - Deployment（包含 HostPath 卷和探针）
  - Service（ClusterIP）
         ↓
设置 OwnerReference（级联删除）
         ↓
更新 VibeApp Status.Conditions
```

---

## 部署架构

```
┌─────────────────────────────────────────────────────┐
│                 Kubernetes Cluster                  │
│                                                     │
│  ┌─────────────────────────────────────────────┐   │
│  │ Namespace: kubebuilder-system               │   │
│  │                                             │   │
│  │  ┌─────────────────────────────────────┐  │   │
│  │  │ Deployment: kubebuilder-vibeapp-    │  │   │
│  │  │ operator-controller-manager         │  │   │
│  │  │  - Container: manager               │  │   │
│  │  │  - Volume Mount: /tmp/k8s-webhook-  │  │   │
│  │  │    server/serving-certs (from      │  │   │
│  │  │    secret: kubebuilder-webhook-     │  │   │
│  │  │    server-cert)                     │  │   │
│  │  └─────────────────────────────────────┘  │   │
│  │                                             │   │
│  │  ┌─────────────────────────────┐          │   │
│  │  │ Service: webhook-service    │          │   │
│  │  │  Port 443 → 9443 (manager)  │          │   │
│  │  └─────────────────────────────┘          │   │
│  │                                             │   │
│  │  ┌─────────────────────────────────────┐  │   │
│  │  │ Secret: kubebuilder-webhook-        │  │   │
│  │  │   server-cert                       │  │   │
│  │  │  - tls.crt                         │  │   │
│  │  │  - tls.key                         │  │   │
│  │  └─────────────────────────────────────┘  │   │
│  │                                             │   │
│  │  Webhook Configurations:                   │   │
│  │  - MutatingWebhookConfiguration           │   │
│  │  - ValidatingWebhookConfiguration         │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
│  ┌─────────────────────────────────────────────┐   │
│  │ Namespace: default (或其他用户命名空间)      │   │
│  │                                             │   │
│  │  ┌─────────────────────────────────────┐  │   │
│  │  │ VibeApp (CR)                       │  │   │
│  │  │  - spec.image: nginx:latest        │  │   │
│  │  │  - spec.replicas: 2                │  │   │
│  │  │  - spec.port: 80                   │  │   │
│  │  └─────────────────────────────────────┘  │   │
│  │           ↳ Owner of ↓                    │   │
│  │  ┌─────────────────────────────────────┐  │   │
│  │  │ Deployment: vibeapp-sample         │  │   │
│  │  │  - Replicas: 2                     │  │   │
│  │  │  - Pods: nginx:latest              │  │   │
│  │  └─────────────────────────────────────┘  │   │
│  │           ↳ Owner of ↓                    │   │
│  │  ┌─────────────────────────────────────┐  │   │
│  │  │ Service: vibeapp-sample            │  │   │
│  │  │  - ClusterIP: 10.96.xxx.xxx        │  │   │
│  │  │  - Port: 80 → 80                   │  │   │
│  │  └─────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

---

## 技术栈

- **语言**: Go 1.20+
- **框架**: Kubebuilder v4.13.1
- **Kubernetes**: 1.35+
- **依赖库**:
  - `sigs.k8s.io/controller-runtime` v0.23.3
  - `k8s.io/api` (apps, core)
  - `k8s.io/apimachinery`
- **工具**:
  - `controller-gen` v0.20.1（代码生成）
  - `kustomize`（部署编排）
  - `ginkgo/gomega`（测试框架）

---

## 配置状态总结

| 组件 | 状态 | 说明 |
|------|------|------|
| API 定义 | ✅ 已配置 | `apps.example.com/v1alpha1` |
| Controller | ✅ 已实现 | 调和 Deployment + Service，自愈逻辑 |
| MutatingWebhook | ✅ 已实现 | 添加标签，设置 replicas 最小值 |
| ValidatingWebhook | ✅ 已实现 | 验证必填字段 |
| TLS 证书 | ✅ 手动管理 | 使用自签名证书，已禁用 cert-manager |
| RBAC | ✅ 自动生成 | 包含所有必要权限 |
| 样本 CR | ✅ 已配置 | `config/samples/` |
| 单元测试 | ✅ 已编写 | controller 和 webhook 测试 |
| 文档 | ✅ 已完善 | README.md、CLAUDE.md |

---

**.md 生成时间**: 2026-03-31  
**.md 版本**: v1.0
