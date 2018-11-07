# Go Register Server

The microservice registration center is implemented by the go programming language, by tightly depend on the Kubernetes, the microservice registration is implemented by monitoring the state changes of the k8s pod, and adapt to the interface of the spring cloud eureka client to fetch service registry. Each microservice fetch  online and healthy micro-services list from the registration center , providing service governance in Choerodon, and sending service up and down events.

## Feature

- [x] service discovery
- [x] send up down event

## Requirements

1. Configuring the file of Kubeclient config
2. Each microservice pod must have the following three labels。

    ```
    choerodon.io/service        (Microservice name)
    choerodon.io/version        (version)
    choerodon.io/metrics-port   (metrics-port)
    ```
  If your service has contextPath, you can specify by `choerodon.io/context-path`

## Installation and Getting Started

```
go run main.go \
--kubeconfig=<kube config file>

```
## Dependencies

- Go 1.9.4 and above
- [Dep](https://github.com/golang/dep)

## Links

* [Change Log](./CHANGELOG.zh-CN.md)

## Contribute

We welcome your input! If you have feedback, please [submit an issue](https://github.com/choerodon/choerodon/issues). If you'd like to participate in development, please read the [documentation of contribution](https://github.com/choerodon/choerodon/blob/master/CONTRIBUTING.md) and submit a pull request.

## Support

If you have any questions and need our support, [reach out to us one way or another](http://choerodon.io/zh/community/).
