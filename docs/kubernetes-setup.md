# Kubernetes Deployment (Roadmap)

> **Note**: This document describes planned functionality. Kubernetes support is on the roadmap.

## Vision

Scale Ollama horizontally across a Kubernetes cluster to handle high-volume task processing with automatic load balancing.

```
                                    ┌─────────────────────────────────────┐
                                    │         Kubernetes Cluster          │
                                    │                                     │
┌──────────────────┐               │  ┌─────────────────────────────┐   │
│  Primary Machine │    HTTP       │  │    Ollama Service (LB)      │   │
│                  │───────────────┼─▶│    ollama.default.svc       │   │
│  bigo CLI        │               │  └──────────────┬──────────────┘   │
│                  │               │                 │                   │
└──────────────────┘               │    ┌────────────┼────────────┐     │
                                    │    ▼            ▼            ▼     │
                                    │ ┌──────┐   ┌──────┐   ┌──────┐   │
                                    │ │Ollama│   │Ollama│   │Ollama│   │
                                    │ │Pod 1 │   │Pod 2 │   │Pod 3 │   │
                                    │ │(GPU) │   │(GPU) │   │(GPU) │   │
                                    │ └──────┘   └──────┘   └──────┘   │
                                    │                                     │
                                    └─────────────────────────────────────┘
```

## Planned Architecture

### Components

1. **Ollama Deployment**: Horizontally scalable pods with GPU access
2. **Model Cache (PVC)**: Shared storage for model files
3. **Load Balancer Service**: Distributes requests across pods
4. **HPA (Horizontal Pod Autoscaler)**: Scale based on GPU utilization

### Draft Manifests

#### Namespace

```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: bigo
```

#### Ollama Deployment

```yaml
# k8s/ollama-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ollama
  namespace: bigo
spec:
  replicas: 3
  selector:
    matchLabels:
      app: ollama
  template:
    metadata:
      labels:
        app: ollama
    spec:
      containers:
      - name: ollama
        image: ollama/ollama:latest
        ports:
        - containerPort: 11434
        env:
        - name: OLLAMA_HOST
          value: "0.0.0.0"
        - name: OLLAMA_MODELS
          value: "/models"
        resources:
          limits:
            nvidia.com/gpu: 1
        volumeMounts:
        - name: model-cache
          mountPath: /models
      volumes:
      - name: model-cache
        persistentVolumeClaim:
          claimName: ollama-models
      nodeSelector:
        gpu: "true"
```

#### Service

```yaml
# k8s/ollama-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: ollama
  namespace: bigo
spec:
  type: LoadBalancer
  ports:
  - port: 11434
    targetPort: 11434
  selector:
    app: ollama
```

#### Persistent Volume for Models

```yaml
# k8s/ollama-pvc.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: ollama-models
  namespace: bigo
spec:
  accessModes:
    - ReadWriteMany
  storageClassName: nfs  # Or your storage class
  resources:
    requests:
      storage: 100Gi
```

#### Horizontal Pod Autoscaler

```yaml
# k8s/ollama-hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: ollama
  namespace: bigo
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: ollama
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: nvidia.com/gpu
      target:
        type: Utilization
        averageUtilization: 80
```

### Model Pre-loading Job

```yaml
# k8s/model-loader-job.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: ollama-model-loader
  namespace: bigo
spec:
  template:
    spec:
      containers:
      - name: loader
        image: ollama/ollama:latest
        command:
        - /bin/sh
        - -c
        - |
          ollama pull phi3:mini-16k
          ollama pull qwen3:8b
          ollama pull qwen3:8b-8k
        env:
        - name: OLLAMA_MODELS
          value: "/models"
        volumeMounts:
        - name: model-cache
          mountPath: /models
      restartPolicy: OnFailure
      volumes:
      - name: model-cache
        persistentVolumeClaim:
          claimName: ollama-models
```

## BigO Configuration for Kubernetes

```yaml
# .bigo/config.yaml
workers:
  ollama:
    enabled: true
    endpoint: http://ollama.bigo.svc.cluster.local:11434  # In-cluster
    # Or external LoadBalancer IP:
    # endpoint: http://ollama.example.com:11434
    max_concurrent: 10  # Higher for cluster
    models:
      fast: phi3:mini-16k
      default: qwen3:8b
      reasoning: qwen3:8b-8k
```

## Planned Features

### Smart Routing

Route to specific pods based on model availability:

```yaml
# Future: Model affinity
workers:
  ollama:
    routing:
      - model: phi3:mini-16k
        pods: [ollama-0, ollama-1]  # Fast models on smaller GPUs
      - model: qwen3:32b
        pods: [ollama-2]  # Large model on big GPU
```

### Request Queuing

Handle burst traffic with Redis queue:

```yaml
# Future: Queue configuration
bus:
  type: redis
  endpoint: redis://redis.bigo.svc:6379
  queue_size: 1000
```

### Multi-Cluster Support

Failover between clusters:

```yaml
# Future: Multi-cluster
workers:
  ollama:
    clusters:
      - name: primary
        endpoint: http://ollama.us-east.example.com:11434
        priority: 1
      - name: secondary
        endpoint: http://ollama.us-west.example.com:11434
        priority: 2
```

## GPU Node Requirements

### NVIDIA GPU Operator

```bash
# Install NVIDIA GPU Operator
helm repo add nvidia https://helm.ngc.nvidia.com/nvidia
helm install gpu-operator nvidia/gpu-operator \
  --namespace gpu-operator \
  --create-namespace
```

### Node Labels

```bash
# Label GPU nodes
kubectl label nodes gpu-node-1 gpu=true
kubectl label nodes gpu-node-2 gpu=true
```

## Monitoring

### Prometheus Metrics

```yaml
# Future: ServiceMonitor
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: ollama
  namespace: bigo
spec:
  selector:
    matchLabels:
      app: ollama
  endpoints:
  - port: metrics
    interval: 30s
```

### Grafana Dashboard

- Requests per second
- GPU utilization per pod
- Model load times
- Queue depth
- Cost per request

## Timeline

| Phase | Feature | Status |
|-------|---------|--------|
| 1 | Basic K8s deployment | Planned |
| 2 | Shared model storage | Planned |
| 3 | HPA based on GPU | Planned |
| 4 | Smart routing | Future |
| 5 | Multi-cluster | Future |

## Contributing

Interested in Kubernetes support? See [CONTRIBUTING.md](../CONTRIBUTING.md) for how to help.
