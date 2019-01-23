package k8s

import (
	"github.com/choerodon/go-register-server/pkg/api/entity"
	"github.com/choerodon/go-register-server/pkg/embed"
	"github.com/choerodon/go-register-server/pkg/utils"
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) QueryConfigMapByName(name string) *v1.ConfigMap {
	if v, ok := c.appNamespace.Load(name); ok {
		configMap, err := c.kubeOperator.CoreV1().ConfigMaps(v.(string)).Get(name, metaV1.GetOptions{})
		if err == nil {
			return configMap
		}
	}
	for _, namespace := range embed.Env.RegisterServiceNamespace {
		configMap, err := c.kubeOperator.CoreV1().ConfigMaps(namespace).Get(name, metaV1.GetOptions{})
		if err == nil {
			c.appNamespace.Store(name, namespace)
			return configMap
		}
	}
	return nil
}

func (c *Controller) CreateConfigMap(dto *entity.SaveConfigDTO) (*v1.ConfigMap, error) {
	configMapOperator := c.kubeOperator.CoreV1().ConfigMaps(dto.Namespace)
	createConfigMap, err := configMapOperator.Create(newV1ConfigMap(dto))
	if err != nil {
		return nil, err
	}
	c.appNamespace.Store(dto.Service, dto.Namespace)
	glog.Infof("Create configMap: %s, namespace: %s success", dto.Service, dto.Namespace)
	return createConfigMap, nil
}

func (c *Controller) UpdateConfigMap(dto *entity.SaveConfigDTO) (*v1.ConfigMap, error) {
	configMapOperator := c.kubeOperator.CoreV1().ConfigMaps(dto.Namespace)
	updateConfigMap, err := configMapOperator.Update(newV1ConfigMap(dto))
	if err != nil {
		return nil, err
	}
	c.appNamespace.Store(dto.Service, dto.Namespace)
	glog.Infof("Update configMap: %s, namespace: %s success", dto.Service, dto.Namespace)
	return updateConfigMap, nil
}

func (c *Controller) QueryConfigMap(name string, namespace string) *v1.ConfigMap {
	configMapOperator := c.kubeOperator.CoreV1().ConfigMaps(namespace)
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
