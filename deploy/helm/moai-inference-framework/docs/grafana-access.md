## Grafana Access and Dashboard Verification

### 1) Port-forward

```bash
kubectl port-forward -n mif services/mif-grafana 3000:80
```

Then open http://localhost:3000

### 2) Admin credentials

- admin-user

```bash
kubectl get secret mif-grafana -n mif -o jsonpath='{.data.admin-user}' | base64 -d && echo
```

- admin-password

```bash
kubectl get secret mif-grafana -n mif -o jsonpath='{.data.admin-password}' | base64 -d && echo
```

### 3) Check dashboards

1. Log in with the above credentials.
2. Left menu → Dashboards → “Browse.”
3. ConfigMap-based dashboards shipped here include `mif-dp`, etc.

### Notes

- Stop port-forward with `Ctrl+C`.
- If you use a different namespace/service/secret name, adjust `-n` and resource names accordingly.
