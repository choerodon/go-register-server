package k8s

import (
	"github.com/choerodon/go-register-server/pkg/api/entity"
	"github.com/choerodon/go-register-server/pkg/embed"
	"github.com/golang/glog"
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) GetConfigMapByName(name string) *v1.ConfigMap {
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

func (c *Controller) CreateOrUpdateConfigMap(dto *entity.CreateConfigDTO) (*v1.ConfigMap, error) {
	configMapOperator := c.kubeOperator.CoreV1().ConfigMaps(dto.Namespace)
	_, err := configMapOperator.Get(dto.Service, metaV1.GetOptions{})
	application := "application"
	if dto.Profile != entity.DefaultProfile {
		application += "-" + dto.Profile
	}
	application += ".yml"
	createOrUpdateConfigMap := &v1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Namespace: dto.Namespace,
			Name:      dto.Service,
			Annotations: map[string]string{entity.ChoerodonService: dto.Service, entity.ChoerodonFeature:
			entity.ChoerodonFeatureConfig, entity.ChoerodonVersion: dto.Version},
		},
		Data: map[string]string{application: dto.Yaml},
	}
	if err == nil {
		updateConfigMap, err := configMapOperator.Update(createOrUpdateConfigMap)
		if err != nil {
			return nil, err
		}
		glog.Infof("Update configMap: %s, namespace: %s success", dto.Service, dto.Namespace)
		return updateConfigMap, nil
	} else {
		createConfigMap, err := configMapOperator.Create(createOrUpdateConfigMap)
		if err != nil {
			return nil, err
		}
		c.appNamespace.Store(dto.Service, dto.Namespace)
		glog.Infof("Create configMap: %s, namespace: %s success", dto.Service, dto.Namespace)
		return createConfigMap, nil
	}
}
