package k8s

import (
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) GetConfigMapByName(name string) *v1.ConfigMap {
	if v, ok := c.appNamespace[name]; ok {
		configMap, err := c.kubeclientset.CoreV1().ConfigMaps(v).Get(name, metaV1.GetOptions{})
		if err == nil {
			return configMap
		}
	}
	for _, namespace := range MonitoringNamespace {
		configMap, err := c.kubeclientset.CoreV1().ConfigMaps(namespace).Get(name, metaV1.GetOptions{})
		if err == nil {
			c.appNamespace[name] = namespace
			return configMap
		}
	}
	return nil
}
