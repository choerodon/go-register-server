package monitor

import (
	"github.com/choerodon/go-register-server/pkg/eureka/apps"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
)

const REGISTER_NAME = "go-register-server"

func GetCpuNum() int {
	return runtime.NumCPU()
}

func GetMemUsage() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return strconv.Itoa(bToMb(m.Sys)) + "MiB"
}

func bToMb(b uint64) int {
	return int(b) / 1024 / 1024
}

func GetIP() string {
	conn, _ := net.Dial("udp", "8.8.8.8:80")
	defer conn.Close()
	localAddr := conn.LocalAddr().String()
	idx := strings.LastIndex(localAddr, ":")
	return localAddr[0:idx]
}

func GetNamespace() (string, []string) {
	listenOn := strings.Split(os.Getenv("REGISTER_SERVICE_NAMESPACE"), ",")
	server := os.Getenv("REGISTER_SERVER_NAMESPACE")
	return server, listenOn
}

func GetGeneralInfo() map[string]interface{} {
	info := make(map[string]interface{})
	server, listenOn := GetNamespace()
	info["NumOfCpu"] = GetCpuNum()
	info["UsedMemory"] = GetMemUsage()
	info["NamespaceOfRegisterServer"] = server
	info["NamespacesOfListeningOn"] = listenOn
	return info
}

func GetInstanceInfo() map[string]interface{} {
	info := make(map[string]interface{})
	info["IpAddr"] = GetIP()
	return info
}

func GetEurekaApplicationInfos(list []*apps.Application) ([]*apps.Instance, []*apps.EurekaInstance) {
	infos := make([]*apps.EurekaInstance, 0)
	var register []*apps.Instance
	for _, value := range list {
		available, inAvailable, availableIds, inAvailableIds := covertApplicationToEurekaInfo(value)
		availableSize := len(available)
		inAvailableSize := len(inAvailable)
		infos = append(infos, &apps.EurekaInstance{
			Name:                 value.Name,
			AMIs:                 strconv.Itoa(availableSize) + "/" + strconv.Itoa(availableSize+inAvailableSize),
			AvailabilityZones:    availableSize,
			Available:            available,
			InAvailable:          inAvailable,
			AvailableInstances:   availableIds,
			InAvailableInstances: inAvailableIds,
		})
		if value.Name == REGISTER_NAME {
			register = available
		}
	}
	return register, infos
}

func covertApplicationToEurekaInfo(application *apps.Application) ([]*apps.Instance, []*apps.Instance, []string, []string) {
	available := make([]*apps.Instance, 0)
	availableIds := make([]string, 0)
	inAvailable := make([]*apps.Instance, 0)
	inAvailableIds := make([]string, 0)
	for _, value := range application.Instances {
		if value.Status == "UP" {
			available = append(available, value)
			availableIds = append(availableIds, value.InstanceId)
		} else {
			inAvailable = append(inAvailable, value)
			inAvailableIds = append(inAvailableIds, value.InstanceId)
		}
	}
	return available, inAvailable, availableIds, inAvailableIds
}

//func covertApplicationToEurekaInfo(application *apps.Application) (int, int, []string, []string) {
//	availableSize := 0
//	inAvailableSize := 0
//	available := "available("
//	inAvailable := "inAvailable("
//	for _, value := range application.Instances {
//		if value.Status == "UP" {
//			availableSize++
//			available = available + value.InstanceId + "、"
//		} else {
//			inAvailableSize++
//			inAvailable = inAvailable + value.InstanceId + "、"
//		}
//	}
//	available = available + ")"
//	inAvailable = inAvailable + ")"
//	return availableSize, inAvailableSize, available, inAvailable
//}
