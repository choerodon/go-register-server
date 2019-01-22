package utils

import (
	"github.com/choerodon/go-register-server/pkg/api/entity"
	"k8s.io/api/core/v1"
	"reflect"
	"strconv"
	"time"
)

type Status struct {
	UP      string
	DOWN    string
	UNKNOWN string
}

const (
	UP              = "UP"
	DOWN            = "DOWN"
	UNKNOWN         = "UNKNOWN"
	dataCentreClass = "com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo"
	dataCentreName  = "MyOwn"
)

func ConvertPod2Instance(pod *v1.Pod) *entity.Instance {

	now := uint64(time.Now().Unix())
	managementPort := pod.Labels[entity.ChoerodonPort]
	serviceName := pod.Labels[entity.ChoerodonService]
	var port int32
	if container := pod.Spec.Containers[0]; len(container.Ports) > 0 {
		port = pod.Spec.Containers[0].Ports[0].ContainerPort
	}
	instanceId := pod.Status.PodIP + ":" + serviceName + ":" + strconv.Itoa(int(port))

	homePage := "http://" + pod.Status.PodIP + ":" + strconv.Itoa(int(port)) + "/"
	statusPageUrl := "http://" + pod.Status.PodIP + ":" + managementPort + "/info"
	healthCheckUrl := "http://" + pod.Status.PodIP + ":" + managementPort + "/health"
	instance := &entity.Instance{
		HostName:         pod.Status.PodIP,
		App:              serviceName,
		IPAddr:           pod.Status.PodIP,
		Status:           UP,
		InstanceId:       instanceId,
		OverriddenStatus: UNKNOWN,
		Port: entity.Port{
			Port:    port,
			Enabled: true,
		},
		SecurePort: entity.Port{
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
		DataCenterInfo: entity.DataCenterInfo{
			Class: dataCentreClass,
			Name:  dataCentreName,
		},
		HomePageUrl:    homePage,
		StatusPageUrl:  statusPageUrl,
		HealthCheckUrl: healthCheckUrl,
	}
	meteData := make(map[string]string)
	meteData["VERSION"] = pod.Labels[entity.ChoerodonVersion]
	contextPath, ok := pod.Labels[entity.ChoerodonContextPathLabel]
	if ok {
		meteData["CONTEXT-PATH"] = contextPath
	}

	instance.Metadata = meteData
	instance.LeaseInfo = entity.LeaseInfo{
		RenewalIntervalInSecs: 10,
		RegistrationTimestamp: now,
		DurationInSecs:        90,
		LastRenewalTimestamp:  now,
		EvictionTimestamp:     0,
		ServiceUpTimestamp:    now,
	}

	return instance
}

// 将递归map转换为简单map
func ConvertRecursiveMapToSingleMap(recursiveMap map[string]interface{}) map[string]interface{} {
	singleMap := make(map[string]interface{})
	recursive(singleMap, "", recursiveMap)
	return singleMap
}

func recursive(singleMap map[string]interface{}, prefix string, recursiveMap map[string]interface{}) {
	for k, v := range recursiveMap {
		var newKey string
		if prefix != "" {
			newKey = prefix + "." + k
		} else {
			newKey = k
		}
		if v == nil {
			continue
		}
		if reflect.TypeOf(v).Kind() == reflect.Map {
			newMap := v.(map[string]interface{})
			recursive(singleMap, newKey, newMap)
		} else {
			singleMap[newKey] = v
		}
	}
}
