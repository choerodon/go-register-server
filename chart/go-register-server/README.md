# Choerodon Asgard Service
Choerodon Asgard Service 是一个任务调度服务，通过`saga` 实现微服务之间的数据一致性。

## Introduction

## Add Helm chart repository

``` bash    
helm repo add choerodon https://openchart.choerodon.com.cn/choerodon/c7n
helm repo update
```

## Installing the Chart

```bash
$ helm install c7n/go-register-server \
      --set service.enabled=true \
      --set service.name=register-server \
      --set env.open.REGISTER_SERVICE_NAMESPACE="c7n-system" \
      --set rbac.create=true \
      --name register-server \
      --namespace c7n-system
```

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`.

## Uninstalling the Chart

```bash
$ helm delete register-server
```

## Configuration

Parameter | Description	| Default
--- |  ---  |  ---  
`replicaCount` | Replicas count | `1`
`deployment.managementPort` | 服务管理端口 | `8000`
`env.open.REGISTER_SERVICE_NAMESPACE` | 注册中心监听的`namespace`，多个`namespace` 用空格间隔 | `c7n-system`
`service.enabled` | 是否创建`service` | `false`
`service.port` | service端口 | `8000`
`service.name` | service名称 | `register-server`
`service.type` | service类型 | `ClusterIP`
`metrics.path` | 收集应用的指标数据路径 | ``
`metrics.group` | 性能指标应用分组 | `go-register-server`
`logs.parser` | 日志收集格式 | `docker`
`ingress.enabled` | 是否创建ingress | `false`
`ingress.host` | ingress地址 | `register.example.com`
`rbac.create` | 是否创建`ClusterRole` 和`serviceAccountName` | `true`
`rbac.serviceAccountName` | serviceAccountName | `default`
`resources.limits` | k8s中容器能使用资源的资源最大值 | `512Mi`
`resources.requests` | k8s中容器使用的最小资源需求 | `256Mi`

## 验证部署
```bash
curl $(kubectl get svc register-server -o jsonpath="{.spec.clusterIP}" -n c7n-system):8000/eureka/apps
```
出现以下类似信息即为成功部署

```json
{
    "name": "go-register-server",
    "instance": [
        {
            "instanceId": "192.168.3.19:go-register-server:8000",
            "hostName": "192.168.3.19",
            "app": "go-register-server",
            "ipAddr": "192.168.3.19",
            "status": "UP",
             ...
             "metadata": {
                "VERSION": "0.18.0"
            },
             ...
        }
    ]
}
```