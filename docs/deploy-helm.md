# Deploying kryptlet with Helm

The kryptlet Helm chart is published to GHCR as an OCI artifact. It requires Helm 3.8 or later.

## Prerequisites

- Helm 3.8+
- `kubectl` access to your cluster
- `age` CLI for encrypting blobs ([installation](https://github.com/FiloSottile/age#installation))

## Install

```bash
helm install kryptlet oci://ghcr.io/thereisnotime/charts/kryptlet \
  --version 0.1.0 \
  --namespace kryptlet \
  --create-namespace
```

The pod starts immediately but serves no blobs until you add some. See [Adding blobs](#adding-blobs) below.

## Adding blobs

### 1. Generate a key pair

```bash
age-keygen -o key.txt
# Public key: age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
```

Keep `key.txt` (the private key) out of git. The public key is safe to share.

### 2. Encrypt your files

```bash
PUBLIC_KEY=$(grep 'public key' key.txt | awk '{print $NF}')

age -r "$PUBLIC_KEY" config.json  > config.age
age -r "$PUBLIC_KEY" secrets.env  > secrets.age
```

### 3. Base64-encode the ciphertext

```bash
base64 -w0 config.age   # paste this value into values.yaml
base64 -w0 secrets.age
```

### 4. Create a values file

```yaml
# kryptlet-values.yaml
blobs:
  config.age: <base64-encoded output from above>
  secrets.age: <base64-encoded output from above>
```

The `.age` files are already encrypted — it is safe to commit them to git.

### 5. Upgrade the release

```bash
helm upgrade kryptlet oci://ghcr.io/thereisnotime/charts/kryptlet \
  --version 0.1.0 \
  --namespace kryptlet \
  -f kryptlet-values.yaml
```

## Query

```bash
PRIVATE_KEY=$(grep 'AGE-SECRET-KEY' key.txt)

curl -H "Authorization: Bearer $PRIVATE_KEY" \
  https://kryptlet.example.com/v1/blob/config
```

The blob name is the filename without the `.age` extension.

## Ingress

Standard Kubernetes Ingress:

```yaml
# kryptlet-values.yaml
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: kryptlet.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: kryptlet-tls
      hosts:
        - kryptlet.example.com
```

For Traefik IngressRoute, disable the built-in ingress and apply a CRD manifest manually:

```yaml
ingress:
  enabled: false
```

```yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: kryptlet
  namespace: kryptlet
spec:
  entryPoints: [websecure]
  routes:
    - match: Host(`kryptlet.example.com`)
      kind: Rule
      services:
        - name: kryptlet
          port: 8080
  tls: {}
```

## All values

| Key | Default | Description |
|-----|---------|-------------|
| `replicaCount` | `1` | Number of replicas |
| `image.repository` | `ghcr.io/thereisnotime/kryptlet` | Image repository |
| `image.tag` | chart `appVersion` | Image tag |
| `image.pullPolicy` | `IfNotPresent` | Pull policy |
| `service.type` | `ClusterIP` | Service type |
| `service.port` | `8080` | Service port |
| `blobDir` | `/etc/kryptlet/blobs` | In-container blob directory |
| `blobs` | `{}` | Map of `filename.age: <base64>` entries |
| `ingress.enabled` | `false` | Create a standard Ingress |
| `resources.requests.cpu` | `10m` | CPU request |
| `resources.requests.memory` | `16Mi` | Memory request |
| `resources.limits.cpu` | `100m` | CPU limit |
| `resources.limits.memory` | `64Mi` | Memory limit |

## Upgrade

```bash
helm upgrade kryptlet oci://ghcr.io/thereisnotime/charts/kryptlet \
  --version <new-version> \
  --namespace kryptlet \
  -f kryptlet-values.yaml
```

## Uninstall

```bash
helm uninstall kryptlet -n kryptlet
kubectl delete namespace kryptlet
```
