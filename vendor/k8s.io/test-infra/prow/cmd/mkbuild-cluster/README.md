# MkBuild-Cluster

The `mkbuild-cluster` program helps create `cluster.yaml` files that [plank] accepts via the `--build-cluster` flag.

This allows prow to run jobs in different clusters than the one where [plank] runs.

See the [getting started] guide for general info about how to configure jobs that target these clusters.

## Usage

Create a new `cluster.yaml` to send to [plank] via `--build-cluster`:

```sh
# Create initial entry
bazel run //prow/cmd/mkbuild-cluster -- \
  --project=P --zone=Z --cluster=C --alias=default --print-entry > cluster.yaml
# Write secret with this entry
kubectl create secret generic build-cluster --from-file=cluster.yaml
```

Now update plank to mount this secret in the container and use the `--build-cluster` flag:

```yaml
spec:
  containers:
  - name: plank
    args:
    - --build-cluster=/etc/cluster/cluster.yaml
    volumeMounts:
    - mountPath: /etc/cluster
      name: cluster
      readOnly: true
  volumes:
  - name: cluster
    secret:
      defaultMode: 420
      secretName: build-cluster
```
Note: restart plank to see the `--build-cluster` flag.

Append additional entries to `cluster.yaml`:

```sh
# Get current values:
kubectl get secrets/build-cluster -o yaml > ~/old.yaml
# Add new value
cat ~/old.yaml | bazel run //prow/cmd/mkbuild-cluster -- \
  --project=P --zone=Z --cluster=C --alias=NEW_CLUSTER \
  > ~/updated.yaml
diff ~/old.yaml ~/updated.yaml
kubectl apply -f ~/updated.yaml
```

Note: restart plank to see the updated values.

## More options:

### Credential errors

By default we validate the new client works before printing out its credentials.

The `--get-client-cert` flag may fix these errors.

On some platform, MasterAuth has [no RBAC permissions](https://github.com/kubernetes/kubernetes/issues/65400) on the server.
If you see an error of the following form, this is likely the case.

```console
Failed: authenticated client could not list pods: response has status "403 Forbidden" and body "{"kind":"Status","apiVersion":"v1","metadata":{},"status":"Failure","message":"pods is forbidden: User \"client\" cannot list pods in the namespace \"kube-system\"","reason":"Forbidden","details":{"kind":"pods"},"code":403}
```

You need to give the user access to pods in that cluster.

```sh
# Create the pod-reader role
kubectl create clusterrole cluster-pod-admin --verb=* --resource=pods
# Give the user access to read pods. The user in this example is 'client'.
kubectl create clusterrolebinding cluster-pod-admin-binding --clusterrole=cluster-pod-admin --user=client
```

### All options

```sh
# Full list of flags like --account, --print-entry, --get-client-cert, etc.
bazel run //prow/cmd/mkbulid-cluster -- --help
```


[getting started]: /prow/getting_started.md
[plank]: /prow/cmd/plank
