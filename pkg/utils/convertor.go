package utils

import (
	"crypto/sha256"
	"fmt"
	"github.com/choerodon/go-register-server/pkg/api/entity"
	"k8s.io/api/core/v1"
	"reflect"
	"time"
)

type Status struct {
	UP      string
	DOWN    string
	UNKNOWN string
}

const (
	UP                = "UP"
	DOWN              = "DOWN"
	UNKNOWN           = "UNKNOWN"
	CUSTOM_APP_PREFIX = "custom"
	DATA_CENTRE_CLASS = "com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo"
	DATA_CENTRE_NAME  = "MyOwn"
)

func ImpInstance(current *entity.Instance, old *entity.Instance) {
	if old == nil {

		if current.Metadata == nil {
			current.Metadata = make(map[string]string, 1)
		}

		current.Metadata["provisioner"] = "custom"
		current.HostName = current.IPAddr
		current.OverriddenStatus = UNKNOWN
		current.CountryId = 8
		current.ActionType = "ADDED"
		current.IsCoordinatingDiscoveryServer = true
		current.SecureVipAddress = current.App
		current.VipAddress = current.App
		current.DataCenterInfo = entity.DataCenterInfo{
			Class: DATA_CENTRE_CLASS,
			Name:  DATA_CENTRE_NAME,
		}
		current.LeaseInfo.DurationInSecs = 90
		current.LeaseInfo.EvictionTimestamp = 0
		current.LeaseInfo.RenewalIntervalInSecs = 10

		current.HomePageUrl = fmt.Sprintf("http://%s:%d/", current.IPAddr, current.Port.Port)
		current.StatusPageUrl = fmt.Sprintf("http://%s:%d/actuator/info", current.IPAddr, current.Port.Port+1)
		current.HealthCheckUrl = fmt.Sprintf("http://%s:%d/actuator/health", current.IPAddr, current.Port.Port+1)

		current.LastUpdatedTimestamp = current.LeaseInfo.RegistrationTimestamp
		current.LeaseInfo.LastRenewalTimestamp = current.LastUpdatedTimestamp
		current.LastDirtyTimestamp = current.LastUpdatedTimestamp
		current.LeaseInfo.ServiceUpTimestamp = current.LastUpdatedTimestamp
	} else {
		old.Status = current.Status

		provisioner := old.Metadata["provisioner"]
		if podSelfLink, ok := old.Metadata["podSelfLink"]; ok {
			current.Metadata["podSelfLink"] = podSelfLink
		}
		old.Metadata = current.Metadata
		old.Metadata["provisioner"] = provisioner

		old.LastUpdatedTimestamp = current.LastUpdatedTimestamp
		old.LeaseInfo.LastRenewalTimestamp = current.LastUpdatedTimestamp
		old.LeaseInfo.ServiceUpTimestamp = current.LastUpdatedTimestamp
	}
}

func ConvertPod2Instance(pod *v1.Pod) *entity.Instance {

	now := uint64(time.Now().Unix())
	managementPort := pod.Labels[entity.ChoerodonPort]
	serviceName := pod.Labels[entity.ChoerodonService]
	var port int32
	if container := pod.Spec.Containers[0]; len(container.Ports) > 0 {
		port = pod.Spec.Containers[0].Ports[0].ContainerPort
	}
	instanceId := fmt.Sprintf("%s:%s:%d", pod.Status.PodIP, serviceName, port)
	homePage := fmt.Sprintf("http://%s:%d/", pod.Status.PodIP, port)
	statusPageUrl := fmt.Sprintf("http://%s:%s/actuator/info", pod.Status.PodIP, managementPort)
	healthCheckUrl := fmt.Sprintf("http://%s:%s/actuator/health", pod.Status.PodIP, managementPort)
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
			Class: DATA_CENTRE_CLASS,
			Name:  DATA_CENTRE_NAME,
		},
		HomePageUrl:    homePage,
		StatusPageUrl:  statusPageUrl,
		HealthCheckUrl: healthCheckUrl,
	}
	meteData := make(map[string]string)
	meteData["provisioner"] = "pod"
	meteData["podSelfLink"] = fmt.Sprintf("%s/%s", pod.GetNamespace(), pod.GetName())
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

func Contain(m map[string]interface{}, k string) bool {
	for mk := range m {
		if mk == k {
			return true
		}
	}
	return false
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

func Sha256(data string) string {
	h := sha256.New()
	h.Write([]byte(data))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func Sha256Map(data map[string]string) string {
	str := ""
	for k, v := range data {
		str = str + k + v
	}
	return Sha256(str)
}
