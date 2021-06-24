## Kubernetes中的ValidatingAdmissionWebhook

本文通过  ValidatingAdmissionWebhook,实现阻止k8s集群使用指定的镜像启动容器

[https://kubernetes.io/zh/docs/reference/access-authn-authz/extensible-admission-controllers/](https://kubernetes.io/zh/docs/reference/access-authn-authz/extensible-admission-controllers/)



ValidatingAdmissionWebhook优势：

- 由于该服务作为Pod运行，因此部署更容易。
- 一切都可以成为kubernetes资源。
- 需要较少的人工干预和访问主机。
- 如果pod或服务不可用，则将允许所有镜像，这在某些情况下会带来安全风险，因此，如果要使用此路径，请确保使其高度可用，这实际上可以通过指定failurePolicytoFail来配置的Ignore（Fail是默认设置）。

ValidatingAdmissionWebhook劣势：

- 具有RBAC权限的任何人都可以更新/更改配置，因为它只是kubernetes的一个资源。

### 实现

由于api-server与webhook之间的交互必须是https,需要先生成证书

1. 创建CRS

```bash
cat <<EOF | cfssl genkey - | cfssljson -bare server
{
  "hosts": [
    "image-blocker-webhook.default.svc", # apiserver 访问webhook的域名格式为:  server_name.namespace.svc
    "image-blocker-webhook.default.svc.cluster.local",
    "image-blocker-webhook.default.pod.cluster.local",
    "192.0.2.24",
    "10.0.34.2"
  ],
  "CN": "system:node:image-blocker-webhook.default.pod.cluster.local",
  "key": {
    "algo": "ecdsa",
    "size": 256
  },
  "names": [
    {
      "O": "system:nodes"
    }
  ]
}
EOF
```

2. 将其应用到集群

```bash
cat <<EOF | kubectl apply -f -
apiVersion: certificates.k8s.io/v1beta1
kind: CertificateSigningRequest
metadata:
  name: image-bouncer-webhook.default
spec:
  request: $(cat server.csr | base64 | tr -d '\n')
  usages:
  - digital signature
  - key encipherment
  - server auth
EOF
```

3. 授权

```bash
kubectl certificate approve image-bouncer-webhook.default
```

4. 生成server.crt

```bash
kubectl get csr image-bouncer-webhook.default -o jsonpath='{.status.certificate}' | base64 --decode > server.crt
```

5. 创建secret

```yaml
kubectl create secret tls tls-image-blocker-webhook --key server-key.pem --cert server.crt
```

接下来使用kubernetes目录中的yaml文件创建ValidatingWebhookConfiguration资源、services 资源,然后将程序编译成二进制并制作镜像,创建对应的webhook pod



最终使用列表中限制的镜像创建pod验证,成功的话,会看到pod不能创建成功,通过api-server的日志可以看到详细的信息