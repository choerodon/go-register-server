package render

import (
	"github.com/choerodon/go-register-server/pkg/k8s"
	"github.com/choerodon/go-register-server/pkg/api/apps"
	"net"
	"os"
	"runtime"
	"strconv"
)

const registerName = "go-register-server"

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
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func GetNamespace() (string, []string) {
	server := os.Getenv("REGISTER_SERVER_NAMESPACE")
	return server, k8s.MonitoringNamespace
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
		available, inAvailable := covertApplicationToEurekaInfo(value)
		availableSize := len(available)
		inAvailableSize := len(inAvailable)
		infos = append(infos, &apps.EurekaInstance{
			Name:              value.Name,
			AMIs:              strconv.Itoa(availableSize) + "/" + strconv.Itoa(availableSize+inAvailableSize),
			AvailabilityZones: availableSize,
			Available:         available,
			InAvailable:       inAvailable,
			AvailableHtml:     InstanceHtml(available),
			InAvailableHtml:   InstanceHtml(inAvailable),
		})
		if value.Name == registerName {
			register = available
		}
	}
	return register, infos
}

func covertApplicationToEurekaInfo(application *apps.Application) ([]*apps.Instance, []*apps.Instance) {
	available := make([]*apps.Instance, 0)
	inAvailable := make([]*apps.Instance, 0)
	for _, value := range application.Instances {
		if value.Status == "UP" {
			available = append(available, value)
		} else {
			inAvailable = append(inAvailable, value)
		}
	}
	return available, inAvailable
}
