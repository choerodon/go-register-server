package k8s

import (
	"github.com/choerodon/go-register-server/pkg/api/repository"
	kubeInformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

var (
	KubeClient          kubernetes.Interface
	KubeInformerFactory kubeInformers.SharedInformerFactory
	AppRepo             *repository.ApplicationRepository
)
