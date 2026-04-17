# Deploying kryptlet with Argo CD

Two options: Helm chart from the OCI registry, or raw manifests tracked in your GitOps repo.

---

## Option 1: Helm chart (recommended)

### Application manifest

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: kryptlet
  namespace: argocd
spec:
  project: default
  source:
    repoURL: ghcr.io/thereisnotime/charts
    chart: kryptlet
    targetRevision: 0.1.0
    helm:
      releaseName: kryptlet
      values: |
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
  destination:
    server: https://kubernetes.default.svc
    namespace: kryptlet
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

Apply it:

```bash
kubectl apply -f kryptlet-application.yaml
argocd app sync kryptlet
argocd app wait kryptlet --health
```

### Adding blobs

Encrypted `.age` files contain only ciphertext and are safe to commit to git. Base64-encode each file and paste the value into the `blobs` map:

```bash
PUBLIC_KEY=$(grep 'public key' key.txt | awk '{print $NF}')
age -r "$PUBLIC_KEY" config.json > config.age
base64 -w0 config.age   # paste this into blobs.config.age
```

### Keeping blobs in a Secret

To avoid putting blob data inline in the Application manifest, create an Argo CD secret and reference it:

```yaml
# Create the Secret (apply manually or via Sealed Secrets / ESO)
kubectl create secret generic kryptlet-blob-values \
  --from-literal=config.age="$(base64 -w0 config.age)" \
  -n kryptlet
```

Then reference it in the Application using an [external values file](https://argo-cd.readthedocs.io/en/stable/user-guide/helm/#values-files) or the Argo CD `valuesObject` with a `secretKeyRef`. For full GitOps, encrypt the Secret with [Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets) or [External Secrets Operator](https://external-secrets.io/) before committing.

---

## Option 2: Raw manifests from your Git repo

Track the kryptlet manifests directly in your GitOps repository and point Argo CD at the directory.

### Directory layout

```
apps/kryptlet/
├── namespace.yaml
├── configmap.yaml
├── deployment.yaml
├── service.yaml
└── kustomization.yaml   (optional — Argo CD can use plain directories too)
```

See [deploy-flux.md](deploy-flux.md) for the full manifest content — the resources are identical.

### Application manifest

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: kryptlet
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/your-org/your-gitops-repo.git
    targetRevision: main
    path: apps/kryptlet
  destination:
    server: https://kubernetes.default.svc
    namespace: kryptlet
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

### Sync

```bash
argocd app create -f kryptlet-application.yaml
argocd app sync kryptlet
argocd app wait kryptlet --health
```

---

## Verify

```bash
kubectl get pods -n kryptlet
kubectl logs -n kryptlet deploy/kryptlet

PRIVATE_KEY=$(grep 'AGE-SECRET-KEY' key.txt)
curl -H "Authorization: Bearer $PRIVATE_KEY" \
  https://kryptlet.example.com/v1/blob/config
```
