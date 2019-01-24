package service

import (
	"github.com/choerodon/go-register-server/pkg/api/entity"
	"github.com/choerodon/go-register-server/pkg/api/metrics"
	"github.com/choerodon/go-register-server/pkg/api/repository"
	"github.com/choerodon/go-register-server/pkg/embed"
	"github.com/emicklei/go-restful"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"html/template"
	"net"
	"runtime"
	"strconv"
	"time"
)

type EurekaPageService interface {
	HomePage(req *restful.Request, resp *restful.Response)
}

type EurekaPageServiceImpl struct {
	appRepo *repository.ApplicationRepository
}

func NewEurekaPageServiceImpl(appRepo *repository.ApplicationRepository) *EurekaPageServiceImpl {
	return &EurekaPageServiceImpl{appRepo: appRepo}
}

func (es *EurekaPageServiceImpl) HomePage(req *restful.Request, resp *restful.Response) {
	metrics.RequestCount.With(prometheus.Labels{"path": req.Request.RequestURI}).Inc()
	t := template.Must(template.ParseFiles("templates/eureka.html"))
	register, eurekaInstances := getEurekaApplicationInfos(es.appRepo.GetApplicationResources().Applications.ApplicationList)
	err := t.Execute(resp.ResponseWriter, &entity.EurekaPage{
		GeneralInfo:        getGeneralInfo(),
		InstanceInfo:       getInstanceInfo(),
		CurrentTime:        time.Now(),
		AvailableRegisters: register,
		EurekaInstances:    eurekaInstances,
	})
	if err != nil {
		glog.Fatalf("Error Get Home Page: %s", err.Error())
	}
}

func instanceHtml(instances []*entity.Instance) template.HTML {
	html := ""
	for i := 0; i < len(instances); i++ {
		html = html + "<a href=\"" + instances[i].StatusPageUrl +
			"\" target=\"_blank\">" + instances[i].InstanceId + "</a>&#12288;"
	}
	return template.HTML(html)
}

func getCpuNum() int {
	return runtime.NumCPU()
}

func getMemUsage() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return strconv.Itoa(bToMb(m.Sys)) + "MiB"
}

func bToMb(b uint64) int {
	return int(b) / 1024 / 1024
}

func getIP() string {
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

func getGeneralInfo() map[string]interface{} {
	info := make(map[string]interface{})
	info["NumOfCpu"] = getCpuNum()
	info["UsedMemory"] = getMemUsage()
	info["NamespaceOfRegisterServer"] = embed.Env.RegisterServerNamespace
	info["NamespacesOfListeningOn"] = embed.Env.RegisterServiceNamespace
	return info
}

func getInstanceInfo() map[string]interface{} {
	info := make(map[string]interface{})
	info["IpAddr"] = getIP()
	return info
}

func getEurekaApplicationInfos(list []*entity.Application) ([]*entity.Instance, []*entity.EurekaInstance) {
	infos := make([]*entity.EurekaInstance, 0)
	var register []*entity.Instance
	for _, value := range list {
		available, inAvailable := covertApplicationToEurekaInfo(value)
		availableSize := len(available)
		inAvailableSize := len(inAvailable)
		infos = append(infos, &entity.EurekaInstance{
			Name:              value.Name,
			AMIs:              strconv.Itoa(availableSize) + "/" + strconv.Itoa(availableSize+inAvailableSize),
			AvailabilityZones: availableSize,
			Available:         available,
			InAvailable:       inAvailable,
			AvailableHtml:     instanceHtml(available),
			InAvailableHtml:   instanceHtml(inAvailable),
		})
		if value.Name == entity.RegisterServerName {
			register = available
		}
	}
	return register, infos
}

func covertApplicationToEurekaInfo(application *entity.Application) ([]*entity.Instance, []*entity.Instance) {
	available := make([]*entity.Instance, 0)
	inAvailable := make([]*entity.Instance, 0)
	for _, value := range application.Instances {
		if value.Status == "UP" {
			available = append(available, value)
		} else {
			inAvailable = append(inAvailable, value)
		}
	}
	return available, inAvailable
}
