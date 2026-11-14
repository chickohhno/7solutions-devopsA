# GKE Monitoring & Observability (LGTM)

## Overview

Centralized monitoring for `hello-api` running on the GKE cluster `hello-api-gke`.

| Tool | Purpose |
| --- | --- |
| Prometheus | Kubernetes node, Pod, and infrastructure metrics |
| Grafana | View metrics, logs, and traces |
| Loki | Store and query application logs |
| Grafana Alloy | Collect Kubernetes Pod logs and forward them to Loki |
| Tempo | Store and query distributed traces |
| OpenTelemetry (Go) | Generate HTTP request traces in `hello-api` |

**Flow:** `GKE metrics → Prometheus → Grafana` · `Pod logs → Alloy → Loki → Grafana` · `HTTP requests → OpenTelemetry → Tempo → Grafana`

## 1. Repository files

The Argo CD Application manifests are in:

```text
argocd/
├── monitoring.yaml   # kube-prometheus-stack (Prometheus, Grafana, Alertmanager)
├── loki.yaml         # Loki log backend
├── alloy.yaml        # Kubernetes log collector
└── tempo.yaml        # Tempo tracing backend
```

All four Applications deploy into namespace `monitoring`. The Go application runs in namespace `production`.

Chart versions used in this project: `kube-prometheus-stack 92.14.1`, `loki 18.7.6`, `alloy 1.15.0`, and `tempo 2.3.0`.

## 2. Deploy the monitoring stack with Argo CD

**Prerequisites:** GKE cluster reachable with `kubectl`; Argo CD already installed in namespace `argocd`.

From the repository root:

```bash
kubectl apply -f argocd/monitoring.yaml
kubectl apply -f argocd/loki.yaml
kubectl apply -f argocd/alloy.yaml
kubectl apply -f argocd/tempo.yaml
```

Check applications and Pods:

```bash
kubectl get applications -n argocd
kubectl get pods -n monitoring
kubectl get svc -n monitoring
```

All Applications should eventually show **Synced / Healthy** and their required Pods should be Ready.

> Application manifests are stored in Git, but these commands bootstrap/update the Argo CD Application objects. To make changes to those objects automatically from Git, manage them with a parent Argo CD Application (app-of-apps).

## 3. Access Grafana

Open a local port-forward (keep the terminal running):

```bash
kubectl port-forward -n monitoring svc/monitoring-grafana 3000:80
```

Open **http://localhost:3000**.

Retrieve the admin credentials:

```bash
kubectl get secret monitoring-grafana -n monitoring -o jsonpath='{.data.admin-user}' | base64 -d; echo
kubectl get secret monitoring-grafana -n monitoring -o jsonpath='{.data.admin-password}' | base64 -d; echo
```

Do not commit credentials to Git. If the password was previously changed inside Grafana, the current database password may differ from the Secret.

## 4. Verify Prometheus metrics

In **Grafana → Explore**, select the **Prometheus** datasource and run:

```promql
up
```

For Pod information:

```promql
kube_pod_status_phase{namespace="production",phase="Running"}
```

You can also view the preinstalled Kubernetes/Node dashboards under **Dashboards**.

**Configuration:** Prometheus retains metrics for `2d` and requests a `10Gi` persistent volume; Grafana requests `5Gi`.

## 5. Connect Loki and verify logs

In **Grafana → Connections → Data sources**, add **Loki** if it is not already configured.

**URL:** `http://loki-gateway.monitoring.svc.cluster.local`

Click **Save & test**.

Alloy collects Pod logs from the `production` namespace and forwards them to Loki.

Create a temporary test Pod:

```bash
kubectl run log-test -n production --image=busybox:1.36 --restart=Never -- sh -c 'echo "hello from alloy to loki"; sleep 300'
```

In **Grafana → Explore → Loki**, query:

```logql
{namespace="production"} |= "hello from alloy to loki"
```

The test message should appear. Clean up:

```bash
kubectl delete pod log-test -n production
```

**Configuration:** Loki uses one monolithic instance with a `10Gi` persistent volume. Alloy runs as a single Deployment and collects logs through the Kubernetes API. Loki is internal to the cluster.

## 6. Connect Tempo and verify tracing

In **Grafana → Connections → Data sources**, add **Tempo** if it is not already configured.

**URL:** `http://tempo.monitoring.svc.cluster.local:3200`

Click **Save & test**.

Tempo receives OpenTelemetry traces over OTLP/HTTP port `4318` (or OTLP/gRPC port `4317`). The Go application is instrumented in `main.go` and uses this environment variable in `k8s/base/deployment.yaml`:

```yaml
env:
  - name: OTEL_EXPORTER_OTLP_ENDPOINT
    value: "http://tempo.monitoring.svc.cluster.local:4318"
```

The production Kustomize overlay currently selects `chickohhno/hello-api:v1.0.2`.

Verify the rollout and live endpoint:

```bash
kubectl rollout status deployment/hello-api -n production
kubectl set env deployment/hello-api -n production --list
kubectl port-forward -n production svc/hello-api 8080:80
```

In another terminal, generate requests:

```bash
for i in {1..10}; do curl -s "http://localhost:8080/?name=GKE-$i"; done
```

Check application logs for a `trace_id`:

```bash
kubectl logs -n production deployment/hello-api --tail=50
```

In **Grafana → Explore → Tempo**, select a time range covering the requests and run:

```traceql
{ resource.service.name = "hello-api" }
```

A successful end-to-end test shows HTTP request traces (including `hello-api.request` and `generate-greeting` spans). A working Tempo datasource alone does **not** prove traces were ingested.

**Configuration:** Tempo uses one replica, `24h` trace retention, and a `10Gi` persistent volume.

## 7. Quick troubleshooting

```bash
# Argo CD sync / health
kubectl get applications -n argocd

# Monitoring components and Services
kubectl get pods,svc -n monitoring

# Collector and trace backend logs
kubectl logs -n monitoring deployment/alloy --tail=50
kubectl logs -n monitoring statefulset/tempo --tail=50

# Application logs and environment
kubectl logs -n production deployment/hello-api --tail=50
kubectl set env deployment/hello-api -n production --list
```

Use the actual resource name shown by `kubectl get deploy,sts -n monitoring` if it differs from the examples above.

## 8. Completion checklist

- [ ] Prometheus returns Kubernetes metrics in Grafana.
- [ ] Loki shows a real Pod log collected by Alloy.
- [ ] Tempo datasource passes **Save & test**.
- [ ] `hello-api` produces a `trace_id` in its logs.
- [ ] Tempo shows an actual `hello-api` HTTP request trace in Grafana.
- [ ] All four monitoring Argo CD Applications are **Synced / Healthy**.

**Scope:** Cost-conscious assessment deployment, not production HA. Loki/Tempo use single instances and persistent disks; public ingress and external object storage are not configured. GKE compute and disk charges continue while resources run.
