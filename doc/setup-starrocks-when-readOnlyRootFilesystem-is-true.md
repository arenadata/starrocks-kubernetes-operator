# Background

StarRocks has three components: Frontend (FE), Backend (BE), and Compute Node (CN). All of them write into
their installation directory by default, which is what `readOnlyRootFilesystem: true` forbids:

| what | FE | BE / CN |
|------|----|---------|
| pid file | `bin/fe.pid` | `bin/{be,cn}.pid` |
| configuration | `conf/fe.conf` | `conf/{be,cn}.conf` |
| temporary files | `temp_dir`, `java.io.tmpdir` | `java.io.tmpdir` of the embedded JVM |
| plugins, small files | `plugins`, `small_files` | `lib/small_file`, `lib/udf`, `lib/udf-runtime` |
| spilled data | - | `spill` |

The operator and the images take care of the first three:

* the configuration is mounted over `conf` and nothing copies it anywhere, so the mount can be read-only
  (a Secret is the recommended source, it usually holds credentials);
* an `emptyDir` is mounted at `/tmp` and `PID_DIR` (plus `UDF_RUNTIME_DIR` for BE and CN) points at it;
* `fsGroup` is set for the pod, so the mounted Secret stays readable for the StarRocks process.

Everything else has to be pointed at a volume in the configuration, see the examples below.

> Note: this needs an image whose entrypoint scripts do not write into the installation directory,
> StarRocks `4.4.0` / `4.0.10.1` or later. With an older image the components either fail to start or
> ignore the mounted configuration. Overriding `command` / `args` (or the `entrypoint` value of the chart)
> is not needed any more and is discouraged: it replaces the ENTRYPOINT of the image, which is what reaps
> the zombie processes of the container.

# By using StarRocksCluster CR yaml

```yaml
apiVersion: starrocks.com/v1
kind: StarRocksCluster
metadata:
  name: kube-starrocks
  namespace: starrocks
spec:
  starRocksFeSpec:
    image: starrocks/fe-ubuntu:latest
    replicas: 1
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    secrets:
    # the configuration is mounted over the conf directory of the installation
    - name: kube-starrocks-fe-conf
      mountPath: /opt/starrocks/fe/conf
    storageVolumes:
    - name: fe-meta
      mountPath: /opt/starrocks/fe/meta
      storageSize: 10Gi
    - name: fe-log
      mountPath: /opt/starrocks/fe/log
      storageSize: 10Gi
    # temp_dir, plugins and small_files of the FE, see the configuration below
    - name: fe-tmp
      mountPath: /opt/starrocks/fe/tmp
      storageClassName: emptyDir
      storageSize: 1Gi
  starRocksBeSpec:
    image: starrocks/be-ubuntu:latest
    replicas: 1
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    secrets:
    - name: kube-starrocks-be-conf
      mountPath: /opt/starrocks/be/conf
    storageVolumes:
    - name: be-data
      mountPath: /opt/starrocks/be/storage
      storageSize: 100Gi
    - name: be-log
      mountPath: /opt/starrocks/be/log
      storageSize: 10Gi
    # spill, small files and UDFs of the BE, see the configuration below
    - name: be-tmp
      mountPath: /opt/starrocks/be/tmp
      storageClassName: emptyDir
      storageSize: 10Gi

---

apiVersion: v1
kind: Secret
metadata:
  name: kube-starrocks-fe-conf
  namespace: starrocks
type: Opaque
stringData:
  fe.conf: |
    LOG_DIR = ${STARROCKS_HOME}/log
    DATE = "$(date +%Y%m%d-%H%M%S)"
    JAVA_OPTS="-Dlog4j2.formatMsgNoLookups=true -Xmx8192m -XX:+UseG1GC -Xlog:gc*:${LOG_DIR}/fe.gc.log.$DATE:time -XX:ErrorFile=${LOG_DIR}/hs_err_pid%p.log"
    http_port = 8030
    rpc_port = 9020
    query_port = 9030
    edit_log_port = 9010
    sys_log_level = INFO

    # the directories that default into the installation directory
    tmp_dir = /opt/starrocks/fe/tmp/temp_dir
    plugin_dir = /opt/starrocks/fe/tmp/plugins
    small_file_dir = /opt/starrocks/fe/tmp/small_files

---

apiVersion: v1
kind: Secret
metadata:
  name: kube-starrocks-be-conf
  namespace: starrocks
type: Opaque
stringData:
  be.conf: |
    be_port = 9060
    webserver_port = 8040
    heartbeat_service_port = 9050
    brpc_port = 8060
    sys_log_level = INFO

    # the directories that default into the installation directory
    spill_local_storage_dir = /opt/starrocks/be/tmp/spill
    small_file_dir = /opt/starrocks/be/tmp/small_file
    user_function_dir = /opt/starrocks/be/tmp/udf
```

# By using Helm Chart

The chart renders the `config` values into a Secret and mounts it over the conf directory itself, so only
the volumes and the redirected paths are left to declare:

```yaml
starrocks:
  starrocksFESpec:
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    storageSpec:
      name: fe
      storageSize: 10Gi
      logStorageSize: 10Gi
    emptyDirs:
    - name: fe-tmp
      mountPath: /opt/starrocks/fe/tmp
    config: |
      LOG_DIR = ${STARROCKS_HOME}/log
      DATE = "$(date +%Y%m%d-%H%M%S)"
      JAVA_OPTS="-Dlog4j2.formatMsgNoLookups=true -Xmx8192m -XX:+UseG1GC -Xlog:gc*:${LOG_DIR}/fe.gc.log.$DATE:time -XX:ErrorFile=${LOG_DIR}/hs_err_pid%p.log"
      http_port = 8030
      rpc_port = 9020
      query_port = 9030
      edit_log_port = 9010
      sys_log_level = INFO
      tmp_dir = /opt/starrocks/fe/tmp/temp_dir
      plugin_dir = /opt/starrocks/fe/tmp/plugins
      small_file_dir = /opt/starrocks/fe/tmp/small_files
  starrocksBeSpec:
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    storageSpec:
      name: be
      storageSize: 100Gi
      logStorageSize: 10Gi
      # mounted at /opt/starrocks/be/spill, which is where spill_local_storage_dir points by default
      spillStorageSize: 10Gi
    emptyDirs:
    - name: be-tmp
      mountPath: /opt/starrocks/be/tmp
    config: |
      be_port = 9060
      webserver_port = 8040
      heartbeat_service_port = 9050
      brpc_port = 8060
      sys_log_level = INFO
      small_file_dir = /opt/starrocks/be/tmp/small_file
      user_function_dir = /opt/starrocks/be/tmp/udf
```

# Migrating from the old recipe

Earlier versions of this document worked around the missing support by copying the whole installation into
an `emptyDir` at start up (`cp -r /opt/starrocks/* /opt/starrocks-artifacts`) and by pointing
`STARROCKS_ROOT` at the copy. That is not needed any more: remove the `command` / `args` (or `entrypoint`)
override, the artifacts volume and the `STARROCKS_ROOT` environment variable, and move the configuration
into a Secret mounted over the conf directory.
