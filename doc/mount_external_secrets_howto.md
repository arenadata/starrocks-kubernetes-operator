# Mount external secrets

StarRocks Kubernetes Operator mounts Secrets into StarRocks. It does not mount ConfigMaps: the
configuration of a component is a credential of its own, and every file the operator mounts is a Secret,
mounted read-only with mode 0440.

## 1. Mount secrets by StarRocks CRD YAML file

You can specify `secrets` in the corresponding component spec. The following is an example.

```yaml
apiVersion: starrocks.com/v1
kind: StarRocksCluster
metadata:
  name: kube-starrocks
  namespace: kb-system
spec:
  starRocksFeSpec:
    image: "starrocks/fe-ubuntu:latest"
    replicas: 1
    secrets:
      - name: my-secret
        mountPath: /etc/my-secret
  starRocksBeSpec:
    image: "starrocks/be-ubuntu:latest"
    replicas: 1
    secrets:
      - name: my-secret
        mountPath: /etc/my-secret
```

> Note: The `Secret` resources should be available in the kubernetes cluster before enabling this feature.

## 2. Mount secrets by helm chart

By using the Helm chart, you can also mount multiple external secrets into StarRocks. You can specify
`secrets` in the corresponding component spec. The following is an example by using the `kube-starrocks`
Helm chart.

```yaml
starrocks:
  starrocksBeSpec:
    secrets:
      # mount the whole secret `my-secret` to `/etc/my-secret`
      - name: my-secret
        mountPath: /etc/my-secret

  # a secret named `my-secret` will be created with the following content.
  secrets:
  - name: my-secret
    data:
      key.conf: |
        this is the content of the secret
        when mounted, key will be the name of the file
```

## 3. Mount a secret to a subPath by Helm Chart

You can also mount external secrets into StarRocks with a subPath. The following is an example by
using the `kube-starrocks` Helm chart.

```yaml
starrocks:
  starrocksBeSpec:
    secrets:
      # mount the file `key.conf` in secret `my-secret` to `/opt/starrocks/be/conf/key.conf`
      - name: my-secret
        mountPath: /opt/starrocks/be/conf/key.conf
        subPath: key.conf

  # a secret named `my-secret` will be created with the following content.
  secrets:
  - name: my-secret
    data:
      key.conf: |
        this is the content of the secret
        when mounted, key will be the name of the file
```

> Note: the conf directory of a component is already replaced by the configuration Secret of the chart, so
> a second mount under it needs a subPath, and the file it mounts is not part of the configuration the
> operator parses.

## Permissions

Every mounted file gets mode 0440 and belongs to the group the pod declares as `fsGroup`, which the
operator always sets to the group the StarRocks process runs with. A mounted file is therefore never
executable: a script delivered this way has to be run through an interpreter, for example
`command: ["bash"]` with the path of the script in `args`, and not executed directly.
