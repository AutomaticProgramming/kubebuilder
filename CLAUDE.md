# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a **Kubernetes Operator** built with **Kubebuilder v3+** that manages the `VibeApp` custom resource. The Operator automates deployment of application workloads by converting VibeApp CR specifications into production-ready Deployments and Services.

**Custom Resource:**
- **Group:** `apps.example.com`
- **Version:** `v1alpha1`
- **Kind:** `VibeApp`
- **Spec Fields:**
  - `image` (string): Container image
  - `replicas` (int32): Desired replica count (minimum 2)
  - `healthCheckPath` (string): HTTP health check endpoint (e.g., `/healthz`)
  - `storagePath` (string): HostPath volume mount location
  - `port` (int32): Container and Service port

**Key Features:**
- Mutating webhook: Auto-adds `created-by: "wang"` label and ensures `replicas` exists (defaults to 2, enforces minimum 2)
- Validating webhook: Validates required fields (image, port, replicas)
- Reconciliation loop: Creates/manages Deployment and Service resources with owner references
- Self-healing: Prevents manual scaling below 2 replicas
- TLS: Self-signed certificates for webhook communication

---

## Initial Setup

**First-time project initialization (run from repository root):**

1. Initialize Kubebuilder project structure:
```bash
make init CLIENT_APIS=apps.example.com/v1alpha1
```

2. Create the API/v1alpha1 scaffolding:
```bash
make api --group apps.example.com --version v1alpha1 --kind VibeApp
```

3. Update the `config/` kustomize manifests to match the API group/version.

4. Generate code and manifests:
```bash
make generate
make manifests
```

---

## Common Development Commands

**Build & Generate:**
```bash
make generate           # Generate code (Deepcopy, conversion, etc.)
make manifests          # Generate CRD manifests
make install            # Install CRDs to cluster
make uninstall         # Remove CRDs from cluster
make build              # Build the operator binary
```

**Running Locally:**
```bash
make run                # Run operator locally against kubeconfig cluster (default)
make run USE_DOCKER=1   # Run operator in Docker locally
```

**Testing:**
```bash
make test              # Run unit tests
make test-e2e          # Run end-to-end tests (requires cluster)
make test-integration  # Run integration tests with EnvTest
```

**CRD Management:**
```bash
make install           # Install CRDs to the cluster
make uninstall        # Delete CRDs from the cluster
```

**Deployment:**
```bash
make deploy            # Deploy operator to cluster using kustomize
make undeploy          # Remove operator from cluster
```

**Debugging:**
```bash
make docker-build      # Build Docker image
make docker-push       # Push Docker image to registry
```

---

## Code Architecture

### Key Components

**1. VibeApp API (api/v1alpha1/)**
- Defines the `VibeApp` custom resource schema.
- Spec contains: `image`, `replicas`, `healthCheckPath`, `storagePath`, `port`
- Status contains reconciliation status conditions and observed generation.

**2. Controller (controllers/vibesapp_controller.go)**
Implements `Reconciler` interface:
- `Reconcile(ctx, req)`: Main reconciliation loop
- Sets up `SchemeBuilder` and `Manager`
- Watches: VibeApp resources, owned Deployments, owned Services
- Business logic:
  - Create/Update Deployment with HostPath volume, liveness probe
  - Create/Update ClusterIP Service
  - Set owner references for garbage collection
  - Enforce minimum replicas (self-healing against manual kubectl scale)

**3. Webhooks (config/webhook/)**
- `mutatingwebhookconfiguration.yaml`: Applies `created-by` label, sets default replicas=2, enforces min=2
- `validatingwebhookconfiguration.yaml`: Validates required fields
- Webhook server runs alongside the manager (port: 9443 by default)

**4. TLS (certs/ and config/webhook/patches/)**
- Self-signed CA and serving certificates
- `ca-bundle.pem` patched into webhook configurations via kustomize

---

## Important Patterns

### Owner References (Garbage Collection)
Use `controller-runtime` to set owner so Deployments/Services are deleted when VibeApp is deleted:
```go
ownerRef := ownerReferenceTo(vibesapp)
setControllerReference(vibesapp, deployment, scheme)
```

### Status Updates
Update VibeApp status with conditions:
```go
vibesapp.Status.Conditions = append(vibesapp.Status.Conditions, metav1.Condition{
    Type:   "Available",
    Status: metav1.ConditionTrue,
    Reason: "ReconciliationSuccess",
})
```

### Self-healing Logic
In the controller, always compare desired state (from VibeApp spec) with actual state (from owned resources). If drift detected (e.g., replicas < spec.replicas or < 2), patch the resource back to desired state.

---

## Testing

**Unit Tests:** Use `controller-runtime`'s `envtest` framework (assets configured in Makefile).
- Place tests in `controllers/` with `_test.go` suffix.
- Use `fake.Client` for unit tests or real `envtest` cluster for integration tests.

**E2E Tests:** Create sample VibeApp CR in `config/samples/`, install operator, create CR, and verify resources.

---

## Dependencies

- Go 1.20+
- Kubebuilder v3+ (as development tool)
- Kubernetes client-go/controller-runtime libraries
- kustomize (for deployment manifests)

---

## Deployment

1. Build and push Docker image:
```bash
make docker-build IMG=your-registry/vibe-operator:tag
make docker-push IMG=your-registry/vibe-operator:tag
```

2. Update `config/manager/manager.yaml` image tag if needed via kustomize.

3. Deploy:
```bash
make deploy IMG=your-registry/vibe-operator:tag
```

4. Create a sample VibeApp:
```bash
kubectl apply -f config/samples/apps_v1alpha1_vibeapp.yaml
```

---

## Notes

- Webhook requires TLS certificates; `make manifests` bundles the CA bundle.
- Use `make install` to install CRDs before creating VibeApp resources.
- For local development, `make run` uses the current kubeconfig context.
- Logging is configured with `klog` (Kubernetes logging library).
- Leader election is enabled by default (using ConfigMap lease).

---

## Debugging Tips

- Check controller logs: `kubectl logs -n <namespace> deployment/vibe-operator-controller-manager`
- Verify webhook health: `kubectl get validatingwebhookconfigurations,mutatingwebhookconfigurations`
- Inspect VibeApp status: `kubectl get vibeapp <name> -o yaml`
- Check owned resources: `kubectl get deployments,services -l app.kubernetes.io/instance=<vibeapp-name>`
