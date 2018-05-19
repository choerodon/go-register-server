package convertor

import (
	"strconv"
	"time"

	"k8s.io/api/core/v1"

	"github.com/choerodon/go-register-server/pkg/eureka/apps"
)

type Status struct {
	UP      string
	DOWN    string
	UNKNOWN string
}

const (
	ChoerodonServiceLabel = "choerodon.io/service"
	ChoerodonVersionLabel = "choerodon.io/version"
	ChoerodonPortLabel    = "choerodon.io/metrics-port"
	UP                    = "UP"
	DOWN                  = "DOWN"
	UNKNOWN               = "UNKNOWN"
	dataCentreClass       = "com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo"
	dataCentreName        = "MyOwn"
)

func ConvertPod2Instance(pod *v1.Pod) *apps.Instance {

	now := uint64(time.Now().Unix())
	managementPort := pod.Labels[ChoerodonPortLabel]
	serviceName := pod.Labels[ChoerodonServiceLabel]
	var port int32
	if container := pod.Spec.Containers[0]; len(container.Ports) > 0 {
		port = pod.Spec.Containers[0].Ports[0].ContainerPort
	}
	instanceId := pod.Status.PodIP + ":" + serviceName + ":" + strconv.Itoa(int(port))

	homePage := "http://" + pod.Status.PodIP + ":" + strconv.Itoa(int(port)) + "/"
	statusPageUrl := "http://" + pod.Status.PodIP + ":" + managementPort + "/info"
	healthCheckUrl := "http://" + pod.Status.PodIP + ":" + managementPort + "/health"
	instance := &apps.Instance{
		HostName:         pod.Status.PodIP,
		App:              serviceName,
		IPAddr:           pod.Status.PodIP,
		Status:           UP,
		InstanceId:       instanceId,
		OverriddenStatus: UNKNOWN,
		Port: apps.Port{
			Port:    port,
			Enabled: true,
		},
		SecurePort: apps.Port{
			Port:    443,
			Enabled: false,
		},
		CountryId:                     8,
		ActionType:                    "ADDED",
		LastDirtyTimestamp:            now,
		LastUpdatedTimestamp:          now,
		IsCoordinatingDiscoveryServer: true,
		SecureVipAddress:              serviceName,
		VipAddress:                    serviceName,
		DataCenterInfo: apps.DataCenterInfo{
			Class: dataCentreClass,
			Name:  dataCentreName,
		},
		HomePageUrl:    homePage,
		StatusPageUrl:  statusPageUrl,
		HealthCheckUrl: healthCheckUrl,
	}
	metedata := make(map[string]string)
	metedata["VERSION"] = pod.Labels[ChoerodonVersionLabel]
	instance.Metadata = metedata
	instance.LeaseInfo = apps.LeaseInfo{
		RenewalIntervalInSecs: 10,
		RegistrationTimestamp: now,
		DurationInSecs:        90,
		LastRenewalTimestamp:  now,
		EvictionTimestamp:     0,
		ServiceUpTimestamp:    now,
	}

	return instance
}
