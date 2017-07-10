package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/runtime"
	utilruntime "k8s.io/client-go/pkg/util/runtime"
	"k8s.io/client-go/pkg/util/wait"
	"k8s.io/client-go/pkg/util/workqueue"
	"k8s.io/client-go/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func getClientsetOrDie(kubeconfig string) *kubernetes.Clientset {
	// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	return clientset
}

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to a kube config. Only required if out-of-cluster.")
	flag.Parse()
	controller := newPodWatchController(*kubeconfig)
	var stopCh <-chan struct{}
	controller.Run(2, stopCh)
}

type podWatcherCtrl struct {
	kubeClient *kubernetes.Clientset

	podStore      cache.StoreToPodLister // A cache of pods
	podController *cache.Controller      // Watches changes to all pods

	podsQueue workqueue.RateLimitingInterface
}

func printPod(pod *v1.Pod) string {
	// data, _ := json.MarshalIndent(pod, "", " ")
	return fmt.Sprintf("%s::%s %s %s", pod.Namespace, pod.Name, pod.Status.PodIP, pod.Status.Phase)
}

func printPodDetail(pod *v1.Pod) string {
	data, _ := json.MarshalIndent(pod, "", " ")
	return fmt.Sprintf("%s::%s %s %s %s", pod.Namespace, pod.Name, pod.Status.PodIP, pod.Status.Phase, string(data))
}

func (slm *podWatcherCtrl) addPod(obj interface{}) {
	pod := obj.(*v1.Pod)
	fmt.Sprintf("AddPod pod [%s]\n", printPod(pod))

	slm.enqueuePod(pod)
}

func (slm *podWatcherCtrl) updatePod(oldObj, newObj interface{}) {
	// oldPod := oldObj.(*v1.Pod)
	newPod := newObj.(*v1.Pod)

	fmt.Printf("UpdatePod [%s] [%s]\n", printPod(newPod), printPodDetail)
	slm.enqueuePod(newObj)
}

func (slm *podWatcherCtrl) delPod(obj interface{}) {
	pod := obj.(*v1.Pod)
	fmt.Printf("DelPod pod [%s]\n", printPod(pod))

	slm.enqueuePod(pod)
}

func (slm *podWatcherCtrl) enqueuePod(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		fmt.Printf("Couldn't get key for object %+v: %v", obj, err)
		return
	}

	slm.podsQueue.Add(key)
}

func (slm *podWatcherCtrl) podWorker() {
	workFunc := func() bool {
		key, quit := slm.podsQueue.Get()
		if quit {
			return true
		}
		defer slm.podsQueue.Done(key)

		obj, exists, err := slm.podStore.Indexer.GetByKey(key.(string))
		if !exists {
			fmt.Printf("Pod has been deleted %v\n", key)
			return false
		}

		if err != nil {
			fmt.Printf("cannot get pod: %v\n", key)
			return false
		}

		pod := obj.(*v1.Pod)
		fmt.Printf("podWorker process IP: %s\n\n", pod.Status.PodIP)
		return false
	}

	for {
		if quit := workFunc(); quit {
			fmt.Printf("pod worker shutting down")
			return
		}
	}
}

func newPodWatchController(kubeconfig string) *podWatcherCtrl {
	slm := &podWatcherCtrl{
		kubeClient: getClientsetOrDie(kubeconfig),
		podsQueue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "pods"),
	}

	slm.podStore.Indexer, slm.podController = cache.NewIndexerInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				return slm.kubeClient.CoreV1().Pods(api.NamespaceAll).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				return slm.kubeClient.CoreV1().Pods(api.NamespaceAll).Watch(options)
			},
		},
		&v1.Pod{},
		// resync is not needed
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    slm.addPod,
			UpdateFunc: slm.updatePod,
			DeleteFunc: slm.delPod,
		},
		cache.Indexers{},
	)

	return slm
}

// Run begins watching and syncing.
func (slm *podWatcherCtrl) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	fmt.Println("Starting podWatcherCtrl Manager")

	go slm.podController.Run(stopCh)

	// wait for the controller to List. This help avoid churns during start up.
	if !cache.WaitForCacheSync(stopCh, slm.podController.HasSynced) {
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(slm.podWorker, time.Second, stopCh)
	}

	<-stopCh
	fmt.Printf("Shutting down Service Lookup Controller")
	slm.podsQueue.ShutDown()
}
