package controller

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/choerodon/go-register-server/pkg/convertor"
	"github.com/choerodon/go-register-server/pkg/eureka/apps"
	"github.com/choerodon/go-register-server/pkg/eureka/repository"
	"os"
	"strings"
)

const (
	ChoerodonServiceLabel = "choerodon.io/service"
	ChoerodonVersionLabel = "choerodon.io/version"
	ChoerodonPortLabel    = "choerodon.io/metrics-port"
)

type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface

	podsLister corelisters.PodLister
	podsSynced cache.InformerSynced

	// workqueue is a rate limited work queue.
	workqueue workqueue.RateLimitingInterface

	appRepo *repository.ApplicationRepository
}

func NewController(
	kubeclientset kubernetes.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	appRepo *repository.ApplicationRepository) *Controller {

	podInformer := kubeInformerFactory.Core().V1().Pods()

	controller := &Controller{
		kubeclientset: kubeclientset,
		podsLister:    podInformer.Lister(),
		podsSynced:    podInformer.Informer().HasSynced,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Pods"),
		appRepo:       appRepo,
	}

	glog.Info("Setting up event handlers")

	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueuePod,
		UpdateFunc: func(oldObj, newObj interface{}) {
			newPod := newObj.(*corev1.Pod)
			oldPod := oldObj.(*corev1.Pod)
			if newPod.ResourceVersion == oldPod.ResourceVersion {
				return
			}
			controller.enqueuePod(newObj)
		},
		DeleteFunc: controller.enqueuePod,
	})

	return controller
}

func (c *Controller) enqueuePod(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	glog.Info("Starting Pod controller")

	// Wait for the caches to be synced before starting workers
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.podsSynced); !ok {
		glog.Error("failed to wait for caches to sync")
	}

	registerServiceNamespaces := strings.Split(os.Getenv("REGISTER_SERVICE_NAMESPACE"), ",")

	glog.Info("Starting workers")
	// Launch two workers to process Foo resources
	for i := 0; i < 2; i++ {
		go wait.Until(func() {
			for c.processNextWorkItem(registerServiceNamespaces) {
			}
		}, time.Second, stopCh)
	}

	glog.Info("Started workers")
	<-stopCh
	glog.Info("Shutting down workers")
}

func (c *Controller) processNextWorkItem(registerServiceNamespaces []string) bool {
	key, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}
	defer c.workqueue.Done(key)

	forget, err := c.syncHandler(key.(string), registerServiceNamespaces)
	if err == nil {
		if forget {
			c.workqueue.Forget(key)
		}
		return true
	}
	runtime.HandleError(fmt.Errorf("error syncing '%s': %s", key, err.Error()))
	c.workqueue.AddRateLimited(key)

	return true
}

func (c *Controller) syncHandler(key string, registerServiceNamespaces []string) (bool, error) {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	matchNum := 0
	for _, ns := range registerServiceNamespaces {
		if strings.Compare(ns, namespace) == 0 {
			matchNum ++
		}
	}
	if matchNum < 1 {
		return true, nil
	}
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return true, nil
	}

	pod, err := c.podsLister.Pods(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			if ins := c.appRepo.DeleteInstance(key); ins != nil {
				ins.Status = apps.DOWN
				glog.Info(key, " DOWN")
			}
			runtime.HandleError(fmt.Errorf("pod '%s' in work queue no longer exists", key))
			return true, nil
		}

		return false, err
	}

	_, isContainServiceLabel := pod.Labels[ChoerodonServiceLabel]
	_, isContainVersionLabel := pod.Labels[ChoerodonVersionLabel]
	_, isContainPortLabel := pod.Labels[ChoerodonPortLabel]

	if !isContainServiceLabel || !isContainVersionLabel || !isContainPortLabel {
		return true, nil
	}

	if pod.Status.ContainerStatuses == nil {
		return true, nil
	}

	if container := pod.Status.ContainerStatuses[0]; container.Ready && container.State.Running != nil && len(pod.Spec.Containers) > 0 {
		if in := convertor.ConvertPod2Instance(pod); c.appRepo.Register(in, key) {
			ins := *in
			ins.Status = apps.UP
			glog.Info(key, " UP ")
		}

	} else {
		if ins := c.appRepo.DeleteInstance(key); ins != nil {
			ins.Status = apps.DOWN
			glog.Info(key, " DOWN")
		}
	}

	return true, nil
}
