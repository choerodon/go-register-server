# register-server

The microservice registration center is implemented in the go language, by tightly integrating the kubinertes, the microservice registration is implemented by monitoring the state changes of the k8s pod, and pull the interface in the spring cloud eureka client service list. Each microservice pulls the registration center to real-time online and healthy micro-services list, providing service governance in Choerodon, and sending service online and offline events.

## Feature

- [x] When launching multiple registrar instances, the service events of online and down are sent through the competition leader.

## Requirements

1. Configuring the file of Kubeclient config
2. Each micro service pod must have the following three labels。

```
choerodon.io/service        (Microservice name)
choerodon.io/version        (version)
choerodon.io/metrics-port   (metrics-port)
```
3. Need two environment variables `KAFKA ADDRESSES` (the address of kafka), `REGISTER _SERVERNAMESPACE` (the k8s namespace that this application belongs to).

## To get the code

```
git clone https://github.com/choerodon/go-register-server.git
```

## Installation and Getting Started

```
go run main.go \
--kubeconfig=<kube config file>

```
## Dependencies

- golang environment

## Reporting Issues
If you find any shortcomings or bugs, please describe them in the Issue.
    
## How to Contribute
Pull requests are welcome! Follow this link for more information on how to contribute.