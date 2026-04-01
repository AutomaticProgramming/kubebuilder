# VibeApp Kubernetes Operator 开发计划

## 上下文 (Context)

这是一个全新的**绿色项目**,需要从零开始构建一个基于 Kubebuilder v3+ 的 Kubernetes Operator。项目仅有 `guide.md` 设计文档,包含完整的 CRD 规范、Webhook 逻辑、控制循环要求和 TLS 配置说明。

**目标:**
- 实现 `VibeApp` 自定义资源 (CRD)
- 开发控制器,将 VibeApp 转换为 Deployment + Service
- 实现变更 Webhook (自动添加标签,设置默认 replica)
- 实现校验 Webhook (验证必填字段)
- 配置自签名证书用于 Webhook TLS
- 确保自愈能力(防止副本数被手动修改为 < 2)

**技术栈:**
- Kubebuilder v3+
- Go 1.20+
- controller-runtime
- kustomize 部署

---

## 开发阶段规划

### 阶段 1: 项目脚手架初始化 (预计 10-15 分钟)

**目标:** 创建 Kubebuilder 标准项目结构和基础 API 定义

**步骤:**
1. 初始化 Kubebuilder 项目
   ```bash
   kubebuilder init --domain example.com --repo apps.example.com/vibe-operator
   ```
   会创建: `PROJECT`, `go.mod`, `Makefile`, `Dockerfile`, `main.go` 等

2. 创建 API 类型
   ```bash
   kubebuilder create api --group apps.example.com --version v1alpha1 --kind VibeApp
   ```
   交互式提示:
   - `Would you like to create a resource under this API group?` → `y`
   - `Would you like to create a controller for the resource?` → `y` (先创建,后续再实现)
   - `Would you like to create a webhook for the resource?` → `n` (暂不创建,后面单独创建)

   生成: `api/v1alpha1/vibesapp_types.go`, `zz_generated.deepcopy.go`

3. 定义 Spec 结构 (编辑 `vibesapp_types.go`):
   - `image string`
   - `replicas *int32`
   - `healthCheckPath string`
   - `storagePath string`
   - `port int32`
   - 加上 JSON/YAML 标签

4. 定义 Status 结构:
   - `Conditions []metav1.Condition` (可选,用于状态追踪)
   - `ObservedGeneration int64`

5. 生成代码和 CRD 清单
   ```bash
   make generate
   make manifests
   ```
   生成: `config/crd/bases/apps.example.com_v1alpha1_vibeapps.yaml`

**验证:**
- `make test` 应该通过 (虽然暂无逻辑)
- CRD YAML 文件应包含正确的 API group/version/kind

---

### 阶段 2: 控制器实现 (预计 30-40 分钟)

**目标:** 实现 VibeApp 的核心 reconciliation 逻辑

**步骤:**
1. 生成控制器骨架
   ```bash
   kubebuilder create controller --group apps.example.com --version v1alpha1 --kind VibeApp
   ```
   提示:
   - `Would you like to generate the controller under existing API group?` → `y`
   - 如果已存在 API,确认覆盖? → 按需选择

   生成: `controllers/vibesapp_controller.go`

2. 实现 `Reconcile` 方法核心逻辑:

   **a. 获取 VibeApp 实例**
   ```go
   vibesapp := &appsv1alpha1.VibeApp{}
   if err := r.Get(ctx, req.NamespacedName, vibesapp); err != nil {
       return ctlr.ignoreNotFound(err)
   }
   ```

   **b. 创建/更新 Deployment**
   - 定义辅助函数 `deploymentForVibeApp(vibesapp)`:
     - Replicas: 如果 `spec.replicas < 2`, 视为 2 (也用于自愈)
     - 容器镜像: `vibesapp.Spec.Image`
     - 端口: `vibesapp.Spec.Port`
     - 存储卷: `HostPath` 类型,路径 `spec.storagePath`, 挂载到容器 `/data`
     - 存活探针: `HTTPGet`, 路径 `spec.healthCheckPath`, 端口 `spec.port`
   - 调用 `r.Create(ctx, deployment)` 或 `r.Update(ctx, deployment)`
   - 设置 OwnerReference: `ctrl.SetControllerReference(vibesapp, deployment, r.Scheme)`

   **c. 创建/更新 Service**
   - 定义辅助函数 `serviceForVibeApp(vibesapp)`:
     - 类型: `ClusterIP`
     - Selector: `app.kubernetes.io/instance: vibesapp.Name`
     - 端口: `spec.port` → `targetPort: spec.port`

   **d. 更新状态 (可选)**
   - 设置 `Status.Conditions` 表示 Available/Progressing
   - 更新 `ObservedGeneration`

   **e. 自愈逻辑**
   - 如果发现 owned Deployment 的 `replicas` 小于期望值(或小于 2),执行 patch 修正

3. 设置控制器 Watches (在 `SetupWithManager` 中):
   - `For(&appsv1alpha1.VibeApp{})` - 主资源
   - `Owns(&appsv1beta1.Deployment{})` - 关联资源
   - `Owns(&corev1.Service{})` - 关联资源

**验证:**
- 运行 `make test` 确保单元测试通过
- 更新 `config/samples/` 示例文件
- 本地运行 `make run` 观察日志

---

### 阶段 3: Webhooks 实现 (预计 30-40 分钟)

**目标:** 实现准入控制——变更和校验

**步骤:**
1. 生成 Webhook 骨架
   ```bash
   kubebuilder create webhook --group apps.example.com --version v1alpha1 --kind VibeApp
   ```
   交互提示:
   - `Would you like to create mutating webhook?` → `y`
   - `Would you like to create validating webhook?` → `y`
   - `Would you like to generate webhook under existing API group?` → `y`

   生成: `controllers/vibesapp_webhook.go`

2. **实现 Mutating Webhook:**
   - 创建 `DefaultVibeApp` 或 `MutateVibeApp` 函数:
     - 添加标签: `metadata.labels["created-by"] = "wang"` (覆盖或新建)
     - 初始化 `replicas`:
       - 如果字段不存在 → 设置为 2
       - 如果值 < 2 → 强制修改为 2
     - (可选) 设置默认 `healthCheckPath="/healthz"` 或 `storagePath="/data"`
   - 返回 `admissionv1.Mutate` 结果

3. **实现 Validating Webhook:**
   - 创建 `ValidateCreate` 和 `ValidateUpdate` 函数:
     - 验证 `spec.image` 非空 (basic check)
     - 验证 `spec.port` > 0
     - 验证 `spec.replicas` 存在且 >= 2 (mutating webhook 已兜底)
   - 失败时返回 `admissionv1.Invalid` 错误

4. 更新 Webhook 配置 (Kubebuilder 会自动注入默认设置):
   - 确保 `config/default/` 中的 webhook manifests 启用这两个 webhook
   - 检查 `validatingwebhookconfiguration.yaml` 和 `mutatingwebhookconfiguration.yaml`

**验证:**
- 单元测试覆盖 webhook 逻辑
- `make test` 通过
- 示例 CR 被 webhook 正确修改

---

### 阶段 4: TLS 证书配置 (预计 15-20 分钟)

**目标:** 配置 Webhook 所需的自签名证书

**步骤:**
1. 生成自签名证书 (通常 Kubebuilder 已提供脚本):
   ```bash
   make certs
   ```
   或手动:
   ```bash
   openssl genrsa -out webhook.key 2048
   openssl req -new -key webhook.key -subj "/CN=webhook" -out webhook.csr
   openssl x509 -req -in webhook.csr -signkey webhook.key -out webhook.crt
   ```

2. 证书文件位置:
   - `certs/` 目录存放服务端证书
   - `ca.crt` 根证书 (用于 APIServer 信任)

3. 配置 CA Bundle 注入:
   - 使用 kustomize `configMapGenerator` 或 `secretGenerator` 将 `ca.crt` 转为 Base64
   - `config/webhook/patch/` 中创建 `ca-bundle-patch.yaml` 补丁
   - 更新 `config/default/webhookcomponents.yaml` 引用该 Secret

4. 更新 Deployment:
   - `config/manager/manager.yaml` 挂载证书到 Volume
   - manager 的 `--webhook-cert-dir` flag 指向证书目录

**验证:**
- Webhook deployments 有证书 Volume 挂载
- `kubectl get validatingwebhookconfiguration` 显示 `caBundle` 已填充

---

### 阶段 5: 集成测试 (预计 20-30 分钟)

**目标:** 确保完整流程在集群或 EnvTest 中工作

**步骤:**
1. 安装 CRDs 到测试集群:
   ```bash
   make install
   ```

2. 部署 Operator 到集群:
   ```bash
   make deploy IMG=your-registry/vibe-operator:latest
   ```
   或本地运行:
   ```bash
   make run USE_DOCKER=1
   ```

3. 创建示例 VibeApp:
   ```bash
   kubectl apply -f config/samples/apps_v1alpha1_vibeapp.yaml
   ```
   预期:
   - Webhook 添加 `created-by` 标签,设置 `replicas=2`
   - Operator 创建 Deployment 和 Service
   - 检查资源状态: `kubectl get deployments,services -l app.kubernetes.io/instance=<name>`

4. 测试自愈:
   - 手动修改 Deployment replicas 为 1: `kubectl scale deployment <name> --replicas=1`
   - 观察 Operator 日志,确认它在下一次 reconcile 中将副本数恢复为 2

5. 测试无效 CR：
   - 创建缺少 `image` 字段的 VibeApp
   - 验证 ValidatingWebhook 拒绝该请求

**测试覆盖:**
- Controller 逻辑: 修改 controllers/vibesapp_controller_test.go
- Webhook 逻辑: 修改 controllers/vibesapp_webhook_test.go
- 使用 `envtest` 运行集成测试: `make test-integration` (需配置)

---

### 阶段 6: 完善与优化 (预计 10-15 分钟)

**目标:** 提升代码质量、文档和可维护性

**步骤:**
1. 更新 `README.md`:
   - 项目介绍、快速开始、部署说明
   - CRD 使用示例

2. 补充 `config/samples/`:
   - 添加一个完整的、有效的 VibeApp 示例
   - 包含必填字段

3. 代码审查:
   - 确保所有 `errors` 被正确处理和记录
   - 检查 nil 指针风险
   - 使用 `klog` 或 `zap` 日志 (推荐 `zap`/`logr`)

4. 添加 Prometheus metrics (可选):
   - `config/default/` 中开启 metrics 服务

5. 文档: 在控制器代码中添加注释,解释关键逻辑

---

## 关键文件路径 (待创建)

| 文件路径 | 责任 | 状态 |
|---------|------|------|
| `api/v1alpha1/vibesapp_types.go` | VibeApp CRD 定义 | 待生成 |
| `controllers/vibesapp_controller.go` | 核心 reconciliation 逻辑 | 待生成 |
| `controllers/vibesapp_webhook.go` | Webhook 变更/校验逻辑 | 待生成 |
| `config/crd/bases/apps.example.com_v1alpha1_vibeapps.yaml` | CRD manifests | 待生成 |
| `config/default/` | 部署所需 kustomize 配置 | 待生成 |
| `config/samples/apps_v1alpha1_vibeapp.yaml` | 示例 CR | 待生成 |
| `certs/` | TLS 证书文件 | 待生成 |

---

## 验证清单

- [ ] 项目能正常编译: `make build`
- [ ] 单元测试通过: `make test`
- [ ] CRD 能正确安装: `make install`
- [ ] Operator 能运行: `make run`
- [ ] 创建 VibeApp → 自动创建 Deployment + Service
- [ ] 手动缩容到 <2 → Operator 自动恢复
- [ ] Webhook 自动添加 `created-by: "wang"`
- [ ] Webhook 自动设置默认 `replicas=2`
- [ ] 无效 CR 被 ValidatingWebhook 拒绝
- [ ] 删除 VibeApp → Deployment + Service 级联删除
- [ ] Webhook 使用 TLS 证书

---

## 风险与注意事项

1. **TLS 证书管理:** Webhook 需要 CA bundle 填充到 WebhookConfiguration。Kubebuilder 的 `make manifests` 会自动处理,但如果手动修改需注意。
2. **自愈循环:** 避免控制器过于激进地修复副本数,建议使用 `client.Patch()` 而非 `client.Update()` 以避免冲突。
3. **RBAC:** Kubebuilder 会生成 role.yaml,确保包含对 `deployments`, `services`, `vibeapps` 的 CRUD 权限。
4. **测试环境:** 本地测试建议使用 EnvTest (Kubernetes API server 二进制),或使用 `kubebuilder.envtest` go module。

---

## 预计总工时

- 阶段 1 (脚手架): 10-15 min
- 阶段 2 (控制器): 30-40 min
- 阶段 3 (Webhooks): 30-40 min
- 阶段 4 (TLS): 15-20 min
- 阶段 5 (集成测试): 20-30 min
- 阶段 6 (完善): 10-15 min

**总计:** 约 2-2.5 小时 (熟练开发者)
