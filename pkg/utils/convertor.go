package utils

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/choerodon/go-register-server/pkg/api/entity"
	"k8s.io/api/core/v1"
	"reflect"
	"time"
)

func ImpInstance(instance *entity.Instance) {
	now := uint64(time.Now().UnixNano() / 1e6)

	if len(instance.Status) == 0 {
		instance.Status = entity.UP
	}

	if instance.Metadata == nil {
		instance.Metadata = make(map[string]string, 1)
	}

	if _, ok := instance.Metadata["provisioner"]; !ok {
		instance.Metadata["provisioner"] = "custom"
	}

	if len(instance.HomePageUrl) == 0 {
		instance.HomePageUrl = fmt.Sprintf("http://%s:%d/",
			instance.IPAddr, instance.Port.Port)
	}
	if len(instance.StatusPageUrl) == 0 {
		instance.StatusPageUrl = fmt.Sprintf("http://%s:%d/actuator/info",
			instance.IPAddr, instance.Port.Port+1)
	}
	if len(instance.HealthCheckUrl) == 0 {
		instance.HealthCheckUrl = fmt.Sprintf("http://%s:%d/actuator/health",
			instance.IPAddr, instance.Port.Port+1)
	}

	instance.HostName = instance.IPAddr
	instance.OverriddenStatus = entity.UNKNOWN
	instance.CountryId = 8
	instance.ActionType = entity.ADDED
	instance.IsCoordinatingDiscoveryServer = true
	instance.VipAddress = instance.App
	instance.SecureVipAddress = instance.App
	instance.DataCenterInfo = entity.DataCenterInfo{
		Class: entity.DATA_CENTRE_CLASS,
		Name:  entity.DATA_CENTRE_NAME,
	}
	instance.LeaseInfo.DurationInSecs = 90
	instance.LeaseInfo.EvictionTimestamp = 0
	instance.LeaseInfo.RenewalIntervalInSecs = 10

	if instance.LeaseInfo.RegistrationTimestamp == 0 {
		instance.LeaseInfo.RegistrationTimestamp = now
	}
	instance.LeaseInfo.LastRenewalTimestamp = now
	instance.LeaseInfo.ServiceUpTimestamp = now

	instance.LastUpdatedTimestamp = now
	instance.LastDirtyTimestamp = now
}

func ConvertPod2Instance(pod *v1.Pod) *entity.Instance {
	now := uint64(time.Now().UnixNano() / 1e6)
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
		Status:           entity.UP,
		InstanceId:       instanceId,
		OverriddenStatus: entity.UNKNOWN,
		Port: entity.Port{
			Port:    port,
			Enabled: true,
		},
		SecurePort: entity.Port{
			Port:    443,
			Enabled: false,
		},
		CountryId:                     8,
		ActionType:                    entity.ADDED,
		LastDirtyTimestamp:            now,
		LastUpdatedTimestamp:          now,
		IsCoordinatingDiscoveryServer: true,
		SecureVipAddress:              serviceName,
		VipAddress:                    serviceName,
		DataCenterInfo: entity.DataCenterInfo{
			Class: entity.DATA_CENTRE_CLASS,
			Name:  entity.DATA_CENTRE_NAME,
		},
		HomePageUrl:    homePage,
		StatusPageUrl:  statusPageUrl,
		HealthCheckUrl: healthCheckUrl,
	}
	meteData := make(map[string]string)
	meteData["provisioner"] = "pod"
	meteData["pod-self-link"] = fmt.Sprintf("%s/%s", pod.GetNamespace(), pod.GetName())
	meteData["version"] = pod.Labels[entity.ChoerodonVersion]
	contextPath, ok := pod.Labels[entity.ChoerodonContextPathLabel]
	if ok {
		meteData["context-path"] = contextPath
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

func DeepCopyInstance(instance *entity.Instance) (*entity.Instance, error) {
	if bytes, e := json.Marshal(instance); e != nil {
		return nil, e
	} else {
		clone := new(entity.Instance)
		if e := json.Unmarshal(bytes, clone); e != nil {
			return nil, e
		}
		return clone, nil
	}
}
