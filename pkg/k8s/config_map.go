package k8s

import (
	"github.com/choerodon/go-register-server/pkg/api/entity"
	"github.com/choerodon/go-register-server/pkg/api/repository"
	"github.com/choerodon/go-register-server/pkg/embed"
	"github.com/choerodon/go-register-server/pkg/utils"
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	configMapV1 "k8s.io/client-go/informers/core/v1"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

var ConfigMapClient *ConfigMapOperatorImpl

type ConfigMapOperator interface {
	QueryConfigMapByName(name string) *v1.ConfigMap
	CreateConfigMap(dto *entity.SaveConfigDTO) (*v1.ConfigMap, error)
	UpdateConfigMap(dto *entity.SaveConfigDTO) (*v1.ConfigMap, error)
	QueryConfigMap(name string, namespace string) *v1.ConfigMap
	StartMonitor(stopCh <-chan struct{})
}

type ConfigMapOperatorImpl struct {
	appNamespace *sync.Map

	kubeV1Client coreV1.CoreV1Interface

	configMapInformer configMapV1.ConfigMapInformer

	configMapCache *sync.Map

	notify chan string

	appRepo *repository.ApplicationRepository
}

func NewConfigMapOperator() ConfigMapOperator {
	configMapInformer := KubeInformerFactory.Core().V1().ConfigMaps()
	if ConfigMapClient == nil {
		ConfigMapClient = &ConfigMapOperatorImpl{
			configMapInformer: configMapInformer,
			appNamespace:      &sync.Map{},
			kubeV1Client:      KubeClient.CoreV1(),
			configMapCache:    &sync.Map{},
			notify:            make(chan string, 50),
			appRepo:           AppRepo,
		}
		ConfigMapClient.configMapInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				mobj := obj.(*v1.ConfigMap)
				if isMonitorNamespace(mobj.Namespace) && mobj.Annotations[entity.ChoerodonFeature] == entity.ChoerodonFeatureConfig {
					_, ok := ConfigMapClient.configMapCache.Load(mobj.Name)
					if !ok {
						ConfigMapClient.configMapCache.Store(mobj.Name, utils.Sha256Map(mobj.Data))
					}
				}
			},
			UpdateFunc: func(old interface{}, new interface{}) {
				mnewobj := new.(*v1.ConfigMap)
				if isMonitorNamespace(mnewobj.Namespace) && mnewobj.Annotations[entity.ChoerodonFeature] == entity.ChoerodonFeatureConfig {
					sha, ok := ConfigMapClient.configMapCache.Load(mnewobj.Name)
					if ok {
						nsha := utils.Sha256Map(mnewobj.Data)
						if sha != nsha {
							ConfigMapClient.configMapCache.Store(mnewobj.Name, utils.Sha256Map(mnewobj.Data))
							ConfigMapClient.notify <- mnewobj.Name
						}
					} else {
						ConfigMapClient.configMapCache.Store(mnewobj.Name, utils.Sha256Map(mnewobj.Data))
					}

				}
			},
			DeleteFunc: func(obj interface{}) {
				mobj := obj.(*v1.ConfigMap)
				if isMonitorNamespace(mobj.Namespace) && mobj.Annotations[entity.ChoerodonFeature] == entity.ChoerodonFeatureConfig {
					ConfigMapClient.configMapCache.Delete(mobj.Name)
				}
			},
		})
	}
	return ConfigMapClient
}

func isMonitorNamespace(namespace string) bool {
	for _, ns := range embed.Env.RegisterServiceNamespace {
		if strings.Compare(ns, namespace) == 0 {
			return true
		}
	}
	return false
}

func (c *ConfigMapOperatorImpl) StartMonitor(stopCh <-chan struct{}) {
	glog.Info("Starting configMap monitor")
	go func() {
		for {
			if d, ok := <-c.notify; ok {
				glog.Infof("ConfigMap %s Changes detected", d)
				instances := c.appRepo.GetInstancesByService(d)
				go c.notifyRefresh(instances)
			}
		}
	}()
	c.configMapInformer.Informer().Run(stopCh)
	glog.Info("Started configMap monitor")
	<-stopCh
	glog.Info("Shutting down configMap monitor")
}

func (c *ConfigMapOperatorImpl) notifyRefresh(instance []*entity.Instance) {
	if len(instance) < 1 {
		return
	}
	for _, v := range instance {
		noticeUri := "http://" + v.IPAddr + ":" + strconv.Itoa(int(v.Port.Port))
		context := v.Metadata[entity.ChoerodonContextPathLabel]
		if context != "" {
			noticeUri = noticeUri + "/" + context
		}
		noticeUri += "/choerodon/config"
		req, _ := http.NewRequest("PUT", noticeUri, nil)
		req.Header.Add("Authorization", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9."+
			"eyJwYXNzd29yZCI6InVua25vd24gcGFzc3dvcmQiLCJ1c2VybmFtZSI6ImRlZmF1bHQiLCJhdXRob3"+
			"JpdGllcyI6W10sImFjY291bnROb25FeHBpcmVkIjp0cnVlLCJhY2NvdW50Tm9uTG9ja2VkIjp0cnVlL"+
			"CJjcmVkZW50aWFsc05vbkV4cGlyZWQiOnRydWUsImVuYWJsZWQiOnRydWUsInVzZXJJZCI6MCwiZW1h"+
			"aWwiOm51bGwsInRpbWVab25lIjoiQ1RUIiwibGFuZ3VhZ2UiOiJ6aF9DTiIsIm9yZ2FuaXphdGlvbklkI"+
			"joxLCJhZGRpdGlvbkluZm8iOm51bGwsImFkbWluIjpmYWxzZX0.Bw96KnS4ZRyEY-77zIetuObbqcu2LR7J03MqwPS6pLI")
		res, error := http.DefaultClient.Do(req)
		if error == nil && 200 <= res.StatusCode && res.StatusCode < 300 {
			glog.Infof("Notify instance %s refresh config success", v.InstanceId)
		} else {
			glog.Warningf("Notify instance %s refresh config failed", v.InstanceId)
		}
	}
}

func (c *ConfigMapOperatorImpl) QueryConfigMapByName(name string) *v1.ConfigMap {
	if v, ok := c.appNamespace.Load(name); ok {
		configMap, err := c.kubeV1Client.ConfigMaps(v.(string)).Get(name, metaV1.GetOptions{})
		if err == nil {
			return configMap
		}
	}
	for _, namespace := range embed.Env.RegisterServiceNamespace {
		configMap, err := c.kubeV1Client.ConfigMaps(namespace).Get(name, metaV1.GetOptions{})
		if err == nil {
			c.appNamespace.Store(name, namespace)
			return configMap
		}
	}
	return nil
}

func (c *ConfigMapOperatorImpl) CreateConfigMap(dto *entity.SaveConfigDTO) (*v1.ConfigMap, error) {
	configMapOperator := c.kubeV1Client.ConfigMaps(dto.Namespace)
	createConfigMap, err := configMapOperator.Create(newV1ConfigMap(dto))
	if err != nil {
		return nil, err
	}
	c.appNamespace.Store(dto.Service, dto.Namespace)
	glog.Infof("Create configMap: %s, namespace: %s success", dto.Service, dto.Namespace)
	return createConfigMap, nil
}

func (c *ConfigMapOperatorImpl) UpdateConfigMap(dto *entity.SaveConfigDTO) (*v1.ConfigMap, error) {
	configMapOperator := c.kubeV1Client.ConfigMaps(dto.Namespace)
	updateConfigMap, err := configMapOperator.Update(newV1ConfigMap(dto))
	if err != nil {
		return nil, err
	}
	c.appNamespace.Store(dto.Service, dto.Namespace)
	glog.Infof("Update configMap: %s, namespace: %s success", dto.Service, dto.Namespace)
	return updateConfigMap, nil
}

func (c *ConfigMapOperatorImpl) QueryConfigMap(name string, namespace string) *v1.ConfigMap {
	configMapOperator := c.kubeV1Client.ConfigMaps(namespace)
	configMap, err := configMapOperator.Get(name, metaV1.GetOptions{})
	if err == nil && configMap != nil {
		return configMap
	}
	return nil
}

func newV1ConfigMap(dto *entity.SaveConfigDTO) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: dto.Namespace,
			Name:      dto.Service,
			Annotations: map[string]string{entity.ChoerodonService: dto.Service, entity.ChoerodonFeature:
			entity.ChoerodonFeatureConfig, entity.ChoerodonVersion: dto.Version},
		},
		Data: map[string]string{utils.ConfigMapProfileKey(dto.Profile): dto.Yaml},
	}
}
