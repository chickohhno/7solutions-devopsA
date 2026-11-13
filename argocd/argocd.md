# Argo CD GitOps Deployment — Hello API

Deploy the Go `hello-api` application to Google Kubernetes Engine (GKE) using **Argo CD** and **Kustomize**.

## Setup

- **Cluster:** `hello-api-gke` (GKE, `asia-southeast1`)
- **Git repository:** https://github.com/chickohhno/7solutions-devopsA
- **Git branch:** `master`
- **Kustomize overlay:** `k8s/overlays/production`
- **Application namespace:** `production`
- **Container image:** `chickohhno/hello-api:v1.0.1`

## 1. Install Argo CD

Make sure `kubectl` points to the GKE cluster:

```bash
kubectl config current-context
kubectl get nodes
```

Install Argo CD:

```bash
kubectl create namespace argocd
kubectl apply -n argocd --server-side --force-conflicts \
  -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
kubectl rollout status deployment/argocd-server -n argocd --timeout=300s
```

> For a long-lived production environment, pin the Argo CD installation to a tested release instead of `stable`.

## 2. Configure the Application

Create `argocd/application.yaml`:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: hello-api
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/chickohhno/7solutions-devopsA.git
    targetRevision: master
    path: k8s/overlays/production
  destination:
    server: https://kubernetes.default.svc
    namespace: production
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

Commit this file to Git for a reproducible, declarative setup. Argo CD reads the Kustomize overlay and synchronizes the generated Kubernetes resources.

## 3. Deploy

```bash
kubectl apply -f argocd/application.yaml
kubectl get applications -n argocd
kubectl get pods,svc -n production
```

The expected Argo CD status is **Synced / Healthy**.

## 4. Verify the API

In one terminal:

```bash
kubectl port-forward -n production svc/hello-api 8080:80
```

In another terminal:

```bash
curl 'http://localhost:8080/?name=GKE'
```

Expected response:

```text
Hello GKE
```

## 5. Access Argo CD UI

```bash
kubectl port-forward -n argocd svc/argocd-server 8081:443
```

Open https://localhost:8081 and sign in as `admin`. Retrieve the initial password locally:

```bash
kubectl get secret argocd-initial-admin-secret -n argocd \
  -o jsonpath='{.data.password}' | base64 -d; echo
```

Change the initial password after first login. Do not commit credentials to Git.

## Updating the Application

1. Push a **new, versioned** Docker image (for example `v1.0.2`).
2. Update `images[].newTag` in `k8s/overlays/production/kustomization.yaml`.
3. Commit and push to `master`.
4. Argo CD detects the Git change and automatically updates the Deployment.

**Note:** Reusing the same image tag does not create a Git manifest change or reliably trigger a rollout.
