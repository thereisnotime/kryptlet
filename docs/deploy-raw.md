# Deploying kryptlet with raw manifests

Use this if you want to deploy kryptlet directly with `kubectl` without a package manager.

## Prerequisites

- `kubectl` access to your cluster
- `age` CLI ([installation](https://github.com/FiloSottile/age#installation))

---

## Quickstart (minimal)

```bash
kubectl create namespace kryptlet

# Generate a key pair
age-keygen -o key.txt
PUBLIC_KEY=$(grep 'public key' key.txt | awk '{print $NF}')

# Encrypt your file
age -r "$PUBLIC_KEY" config.json > config.age

# Create the ConfigMap with the encrypted blob
kubectl create configmap kryptlet-blobs \
  --from-file=config.age \
  -n kryptlet

# Deploy kryptlet
kubectl apply -f - <<'EOF'
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
      containers:
        - name: kryptlet
          image: ghcr.io/thereisnotime/kryptlet:latest
          ports:
            - containerPort: 8080
          env:
            - name: KRYPTLET_BLOB_DIR
              value: /etc/kryptlet/blobs
          volumeMounts:
            - name: blobs
              mountPath: /etc/kryptlet/blobs
              readOnly: true
          livenessProbe:
            httpGet: {path: /healthz, port: 8080}
          readinessProbe:
            httpGet: {path: /readyz, port: 8080}
      volumes:
        - name: blobs
          configMap:
            name: kryptlet-blobs
---
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
      targetPort: 8080
EOF
```

### Query

```bash
PRIVATE_KEY=$(grep 'AGE-SECRET-KEY' key.txt)

kubectl port-forward -n kryptlet svc/kryptlet 8080:8080 &
curl -H "Authorization: Bearer $PRIVATE_KEY" http://localhost:8080/v1/blob/config
```

---

## Production-ready manifests

The manifests below include security hardening: non-root user, read-only filesystem, dropped capabilities, resource limits, and seccomp profile.

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
# Add encrypted blobs as binaryData entries.
# Each key must end in .age and the value must be base64-encoded.
# Generate: age -r <pubkey> myfile > myfile.age && base64 -w0 myfile.age
binaryData:
  config.age: <base64-encoded age ciphertext>
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
              protocol: TCP
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
      name: http
```

### ingress.yaml (nginx)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: kryptlet
  namespace: kryptlet
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  ingressClassName: nginx
  rules:
    - host: kryptlet.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: kryptlet
                port:
                  name: http
  tls:
    - secretName: kryptlet-tls
      hosts:
        - kryptlet.example.com
```

### Apply

```bash
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f ingress.yaml   # if needed
```

### Verify

```bash
kubectl get pods -n kryptlet
kubectl logs -n kryptlet deploy/kryptlet

PRIVATE_KEY=$(grep 'AGE-SECRET-KEY' key.txt)
curl -H "Authorization: Bearer $PRIVATE_KEY" \
  https://kryptlet.example.com/v1/blob/config
```

---

## Updating blobs

Edit `configmap.yaml` with the new base64-encoded blob and re-apply. Kubernetes will roll out a new pod automatically if you set `spec.template.metadata.annotations` with a hash of the ConfigMap, or use a tool like [reloader](https://github.com/stakater/Reloader):

```yaml
metadata:
  annotations:
    configmap.reloader.stakater.com/reload: kryptlet-blobs
```

Or trigger a rollout manually:

```bash
kubectl rollout restart deployment/kryptlet -n kryptlet
```
