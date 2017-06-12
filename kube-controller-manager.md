# kube-controller-manager 源码分析

## 1. Run 函数入口

k8s.io/kubernetes/cmd/kube-controller-manager/controller-manager.go

```go
func main() {
	s := options.NewCMServer()
	s.AddFlags(pflag.CommandLine, app.KnownControllers(), app.ControllersDisabledByDefault.List())

	flag.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()

	verflag.PrintAndExitIfRequested()

	if err := app.Run(s); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
```

k8s.io/kubernetes/cmd/kube-controller-manager/app/controllermanager.go


```go
// Run runs the CMServer.  This should never exit.
func Run(s *options.CMServer) error {
	// ...

	run := func(stop <-chan struct{}) {
	// ...

		err := StartControllers(NewControllerInitializers(), s, rootClientBuilder, clientBuilder, stop)

	leaderelection.RunOrDie(leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: s.LeaderElection.LeaseDuration.Duration,
		RenewDeadline: s.LeaderElection.RenewDeadline.Duration,
		RetryPeriod:   s.LeaderElection.RetryPeriod.Duration,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: run,  // call run
			OnStoppedLeading: func() {
				glog.Fatalf("leaderelection lost")
			},
		},
	})
	}
	
	// ...

```

```go
// NewControllerInitializers is a public map of named controller groups (you can start more than one in an init func)
// paired to their InitFunc.  This allows for structured downstream composition and subdivision.
func NewControllerInitializers() map[string]InitFunc {
	controllers := map[string]InitFunc{}
	controllers["endpoint"] = startEndpointController
	controllers["replicationcontroller"] = startReplicationController
	// ....
	controllers["tokencleaner"] = startTokenCleanerController

	return controllers
}

func StartControllers(controllers map[string]InitFunc, s *options.CMServer, rootClientBuilder, clientBuilder controller.ControllerClientBuilder, stop <-chan struct{}) error {
	sharedInformers := informers.NewSharedInformerFactory(versionedClient, ResyncPeriod(s)())

	ctx := ControllerContext{
		ClientBuilder:      clientBuilder,
		InformerFactory:    sharedInformers,
		Options:            *s,
		AvailableResources: availableResources,
		Stop:               stop,
	}
	
	// ...
	for controllerName, initFn := range controllers {
		// ...
		started, err := initFn(ctx)
	}

	// all the remaining plugins want this cloud variable
	cloud, err := cloudprovider.InitCloudProvider(s.CloudProvider, s.CloudConfigFile)
	
	// ...
	sharedInformers.Start(stop)

	select {}
}

```

## 2. ReplicationController 流程分析

k8s.io/kubernetes/cmd/kube-controller-manager/app/core.go

```go
/*
	sharedInformers := informers.NewSharedInformerFactory(versionedClient, ResyncPeriod(s)())

	ctx := ControllerContext{
		ClientBuilder:      clientBuilder,
		InformerFactory:    sharedInformers,
		Options:            *s,
		AvailableResources: availableResources,
		Stop:               stop,
	}
*/
	
func startReplicationController(ctx ControllerContext) (bool, error) {
	go replicationcontroller.NewReplicationManager(
		ctx.InformerFactory.Core().V1().Pods(),
		ctx.InformerFactory.Core().V1().ReplicationControllers(),
		ctx.ClientBuilder.ClientOrDie("replication-controller"),
		replicationcontroller.BurstReplicas,
	).Run(int(ctx.Options.ConcurrentRCSyncs), ctx.Stop)
	return true, nil
}
```

k8s.io/kubernetes/pkg/controller/replication/replication_controller.go

```go
// NewReplicationManager configures a replication manager with the specified event recorder
func NewReplicationManager(podInformer coreinformers.PodInformer, rcInformer coreinformers.ReplicationControllerInformer, kubeClient clientset.Interface, burstReplicas int) *ReplicationManager {
	if kubeClient != nil && kubeClient.Core().RESTClient().GetRateLimiter() != nil {
		metrics.RegisterMetricAndTrackRateLimiterUsage("replication_controller", kubeClient.Core().RESTClient().GetRateLimiter())
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: v1core.New(kubeClient.Core().RESTClient()).Events("")})

	rm := &ReplicationManager{
		kubeClient: kubeClient,
		podControl: controller.RealPodControl{
			KubeClient: kubeClient,
			Recorder:   eventBroadcaster.NewRecorder(api.Scheme, clientv1.EventSource{Component: "replication-controller"}),
		},
		burstReplicas: burstReplicas,
		expectations:  controller.NewUIDTrackingControllerExpectations(controller.NewControllerExpectations()),
		queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "replicationmanager"),
	}

	rcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    rm.enqueueController,
		UpdateFunc: rm.updateRC,
		// This will enter the sync loop and no-op, because the controller has been deleted from the store.
		// Note that deleting a controller immediately after scaling it to 0 will not work. The recommended
		// way of achieving this is by performing a `stop` operation on the controller.
		DeleteFunc: rm.enqueueController,
	})
	rm.rcLister = rcInformer.Lister()
	rm.rcListerSynced = rcInformer.Informer().HasSynced

	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: rm.addPod,
		// This invokes the rc for every pod change, eg: host assignment. Though this might seem like overkill
		// the most frequent pod update is status, and the associated rc will only list from local storage, so
		// it should be ok.
		UpdateFunc: rm.updatePod,
		DeleteFunc: rm.deletePod,
	})
	rm.podLister = podInformer.Lister()
	rm.podListerSynced = podInformer.Informer().HasSynced

	rm.syncHandler = rm.syncReplicationController
	return rm
}

// ReplicationManager is responsible for synchronizing ReplicationController objects stored
// in the system with actual running pods.
// NOTE: using this name to distinguish this type from API object "ReplicationController"; will
//       not fix it right now. Refer to #41459 for more detail.
type ReplicationManager struct {
	kubeClient clientset.Interface
	podControl controller.PodControlInterface

	// An rc is temporarily suspended after creating/deleting these many replicas.
	// It resumes normal action after observing the watch events for them.
	burstReplicas int
	// To allow injection of syncReplicationController for testing.
	syncHandler func(rcKey string) error

	// A TTLCache of pod creates/deletes each rc expects to see.
	expectations *controller.UIDTrackingControllerExpectations

	rcLister       corelisters.ReplicationControllerLister
	rcListerSynced cache.InformerSynced

	podLister corelisters.PodLister
	// podListerSynced returns true if the pod store has been synced at least once.
	// Added as a member to the struct to allow injection for testing.
	podListerSynced cache.InformerSynced

	// Controllers that need to be synced
	queue workqueue.RateLimitingInterface
}

// Run begins watching and syncing.
func (rm *ReplicationManager) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer rm.queue.ShutDown()

	glog.Infof("Starting RC controller")
	defer glog.Infof("Shutting down RC controller")

	// 等待 所有的时间已经同步
	if !controller.WaitForCacheSync("RC", stopCh, rm.podListerSynced, rm.rcListerSynced) {
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(rm.worker, time.Second, stopCh)
	}

	<-stopCh
}


// worker runs a worker thread that just dequeues items, processes them, and marks them done.
// It enforces that the syncHandler is never invoked concurrently with the same key.
func (rm *ReplicationManager) worker() {
	for rm.processNextWorkItem() {
	}
	glog.Infof("replication controller worker shutting down")
}

func (rm *ReplicationManager) processNextWorkItem() bool {
	key, quit := rm.queue.Get()
	if quit {
		return false
	}
	defer rm.queue.Done(key)

	// rm.syncReplicationController
	err := rm.syncHandler(key.(string))
	if err == nil {
		rm.queue.Forget(key)
		return true
	}

	rm.queue.AddRateLimited(key)
	utilruntime.HandleError(err)
	return true
}

// syncReplicationController will sync the rc with the given key if it has had its expectations fulfilled, meaning
// it did not expect to see any more of its pods created or deleted. This function is not meant to be invoked
// concurrently with the same key.
func (rm *ReplicationManager) syncReplicationController(key string) error {
	trace := utiltrace.New("syncReplicationController: " + key)
	defer trace.LogIfLong(250 * time.Millisecond)

	startTime := time.Now()
	defer func() {
		glog.V(4).Infof("Finished syncing controller %q (%v)", key, time.Now().Sub(startTime))
	}()

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	rc, err := rm.rcLister.ReplicationControllers(namespace).Get(name)
	if errors.IsNotFound(err) {
		glog.Infof("Replication Controller has been deleted %v", key)
		rm.expectations.DeleteExpectations(key)
		return nil
	}
	if err != nil {
		return err
	}

	trace.Step("ReplicationController restored")
	rcNeedsSync := rm.expectations.SatisfiedExpectations(key)
	trace.Step("Expectations restored")

	// list all pods to include the pods that don't match the rc's selector
	// anymore but has the stale controller ref.
	// TODO: Do the List and Filter in a single pass, or use an index.
	allPods, err := rm.podLister.Pods(rc.Namespace).List(labels.Everything())
	if err != nil {
		return err
	}
	// Ignore inactive pods.
	var filteredPods []*v1.Pod
	for _, pod := range allPods {
		if controller.IsPodActive(pod) {
			filteredPods = append(filteredPods, pod)
		}
	}
	// If any adoptions are attempted, we should first recheck for deletion with
	// an uncached quorum read sometime after listing Pods (see #42639).
	canAdoptFunc := controller.RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh, err := rm.kubeClient.CoreV1().ReplicationControllers(rc.Namespace).Get(rc.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if fresh.UID != rc.UID {
			return nil, fmt.Errorf("original ReplicationController %v/%v is gone: got uid %v, wanted %v", rc.Namespace, rc.Name, fresh.UID, rc.UID)
		}
		return fresh, nil
	})
	cm := controller.NewPodControllerRefManager(rm.podControl, rc, labels.Set(rc.Spec.Selector).AsSelectorPreValidated(), controllerKind, canAdoptFunc)
	// NOTE: filteredPods are pointing to objects from cache - if you need to
	// modify them, you need to copy it first.
	filteredPods, err = cm.ClaimPods(filteredPods)
	if err != nil {
		return err
	}

	var manageReplicasErr error
	if rcNeedsSync && rc.DeletionTimestamp == nil {
		manageReplicasErr = rm.manageReplicas(filteredPods, rc)
	}
	trace.Step("manageReplicas done")

	copy, err := api.Scheme.DeepCopy(rc)
	if err != nil {
		return err
	}
	rc = copy.(*v1.ReplicationController)

	newStatus := calculateStatus(rc, filteredPods, manageReplicasErr)

	// Always updates status as pods come up or die.
	updatedRC, err := updateReplicationControllerStatus(rm.kubeClient.Core().ReplicationControllers(rc.Namespace), *rc, newStatus)
	if err != nil {
		// Multiple things could lead to this update failing.  Returning an error causes a requeue without forcing a hotloop
		return err
	}
	// Resync the ReplicationController after MinReadySeconds as a last line of defense to guard against clock-skew.
	if manageReplicasErr == nil && updatedRC.Spec.MinReadySeconds > 0 &&
		updatedRC.Status.ReadyReplicas == *(updatedRC.Spec.Replicas) &&
		updatedRC.Status.AvailableReplicas != *(updatedRC.Spec.Replicas) {
		rm.enqueueControllerAfter(updatedRC, time.Duration(updatedRC.Spec.MinReadySeconds)*time.Second)
	}
	return manageReplicasErr
}

```