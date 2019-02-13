package k8s

import (
	"fmt"
	"github.com/choerodon/go-register-server/pkg/api/entity"
	"github.com/choerodon/go-register-server/pkg/embed"
	"github.com/choerodon/go-register-server/pkg/utils"
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"sync"
)

var ConfigMapClient ConfigMapOperator

type ConfigMapOperator interface {
	QueryConfigMapByName(name string) *v1.ConfigMap
	CreateConfigMap(dto *entity.SaveConfigDTO) (*v1.ConfigMap, error)
	UpdateConfigMap(dto *entity.SaveConfigDTO) (*v1.ConfigMap, error)
	QueryConfigMap(name string, namespace string) *v1.ConfigMap
	StartMonitor()
}

func NewConfigMapOperator() ConfigMapOperator {
	if ConfigMapClient == nil {
		ConfigMapClient = &ConfigMapOperatorImpl{
			appNamespace: &sync.Map{},
			kubeV1Client: KubeClient.CoreV1(),
		}
	}
	return ConfigMapClient
}

type ConfigMapOperatorImpl struct {
	appNamespace *sync.Map
	kubeV1Client coreV1.CoreV1Interface
}

func (c *ConfigMapOperatorImpl) StartMonitor() {
  KubeInformerFactory.Core().V1().ConfigMaps().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
	  UpdateFunc: c.ConfigMapUpdateEvent,
  })
}

func (c *ConfigMapOperatorImpl) ConfigMapUpdateEvent(oldObj, newObj interface{}) {
   fmt.Println(oldObj, newObj)

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
