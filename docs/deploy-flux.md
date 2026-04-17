# Deploying kryptlet with Flux CD

Two approaches are covered here:

- **HelmRelease** (recommended) — uses the published OCI chart; blobs live in `values` or a `valuesFrom` Secret
- **Kustomize** — plain manifests checked into your GitOps repo; useful if you manage everything as YAML

---

## Approach 1: HelmRelease

### OCIRepository + HelmRelease

```yaml
# kryptlet/ocirepository.yaml
apiVersion: source.toolkit.fluxcd.io/v1beta2
kind: OCIRepository
metadata:
  name: kryptlet
  namespace: kryptlet
spec:
  interval: 1h
  url: oci://ghcr.io/thereisnotime/charts/kryptlet
  ref:
    tag: 0.1.0
```

```yaml
# kryptlet/helmrelease.yaml
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: kryptlet
  namespace: kryptlet
spec:
  interval: 1h
  chartRef:
    kind: OCIRepository
    name: kryptlet
    namespace: kryptlet
  values:
    replicaCount: 1
    ingress:
      enabled: true
      className: nginx
      hosts:
        - host: kryptlet.example.com
          paths:
            - path: /
              pathType: Prefix
    blobs:
      config.age: <base64-encoded age ciphertext>
      secrets.age: <base64-encoded age ciphertext>
```

```yaml
# kryptlet/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: kryptlet
```

```yaml
# kryptlet/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - namespace.yaml
  - ocirepository.yaml
  - helmrelease.yaml
```

### Adding blobs

Encrypted `.age` files are safe to commit — they contain only ciphertext. Base64-encode each one and add it to `values.blobs`:

```bash
PUBLIC_KEY=$(grep 'public key' key.txt | awk '{print $NF}')
age -r "$PUBLIC_KEY" config.json > config.age
echo "  config.age: $(base64 -w0 config.age)"
```

Paste the output into the `blobs:` map in your HelmRelease and commit.

### Keeping blob values in a Secret

If you prefer not to put blob data in the HelmRelease manifest, use `valuesFrom`:

```yaml
# kryptlet/blobs-secret.yaml  (apply manually — do not commit plaintext)
apiVersion: v1
kind: Secret
metadata:
  name: kryptlet-blob-values
  namespace: kryptlet
stringData:
  values.yaml: |
    blobs:
      config.age: <base64>
      secrets.age: <base64>
```

```yaml
# in HelmRelease spec:
spec:
  valuesFrom:
    - kind: Secret
      name: kryptlet-blob-values
```

Or use [SOPS](https://fluxcd.io/flux/guides/mozilla-sops/) to encrypt the Secret before committing.

---

## Approach 2: Kustomize

Wire kryptlet as a plain Kustomize overlay if you prefer managing all resources as YAML.

### Directory layout

```
kryptlet/
├── namespace.yaml
├── configmap.yaml
├── deployment.yaml
├── service.yaml
├── ingress.yaml          (optional)
└── kustomization.yaml
```

### namespace.yaml

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: kryptlet
```

### configmap.yaml

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kryptlet-blobs
  namespace: kryptlet
binaryData:
  config.age: <base64-encoded age ciphertext>
  secrets.age: <base64-encoded age ciphertext>
```

### deployment.yaml

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kryptlet
  namespace: kryptlet
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kryptlet
  template:
    metadata:
      labels:
        app: kryptlet
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 65532
        runAsGroup: 65532
        seccompProfile:
          type: RuntimeDefault
      containers:
        - name: kryptlet
          image: ghcr.io/thereisnotime/kryptlet:latest
          ports:
            - name: http
              containerPort: 8080
          env:
            - name: KRYPTLET_ADDR
              value: ":8080"
            - name: KRYPTLET_BLOB_DIR
              value: /etc/kryptlet/blobs
          volumeMounts:
            - name: blobs
              mountPath: /etc/kryptlet/blobs
              readOnly: true
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop: [ALL]
          livenessProbe:
            httpGet:
              path: /healthz
              port: http
            initialDelaySeconds: 5
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /readyz
              port: http
            initialDelaySeconds: 3
            periodSeconds: 5
          resources:
            requests:
              cpu: 10m
              memory: 16Mi
            limits:
              cpu: 100m
              memory: 64Mi
      volumes:
        - name: blobs
          configMap:
            name: kryptlet-blobs
```

### service.yaml

```yaml
apiVersion: v1
kind: Service
metadata:
  name: kryptlet
  namespace: kryptlet
spec:
  selector:
    app: kryptlet
  ports:
    - port: 8080
      targetPort: http
      protocol: TCP
```

### kustomization.yaml

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - namespace.yaml
  - configmap.yaml
  - deployment.yaml
  - service.yaml
```

### Flux Kustomization CR

Point a Flux `Kustomization` at the directory:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: kryptlet
  namespace: flux-system
spec:
  interval: 10m
  path: ./kryptlet
  prune: true
  sourceRef:
    kind: GitRepository
    name: flux-system
```

### Reconcile

```bash
flux reconcile source git flux-system
flux reconcile kustomization kryptlet
kubectl get pods -n kryptlet
```
