# kube-controller-manager 源码分析

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
	controllers["podgc"] = startPodGCController
	controllers["resourcequota"] = startResourceQuotaController
	controllers["namespace"] = startNamespaceController
	controllers["serviceaccount"] = startServiceAccountController
	controllers["garbagecollector"] = startGarbageCollectorController
	controllers["daemonset"] = startDaemonSetController
	controllers["job"] = startJobController
	controllers["deployment"] = startDeploymentController
	controllers["replicaset"] = startReplicaSetController
	controllers["horizontalpodautoscaling"] = startHPAController
	controllers["disruption"] = startDisruptionController
	controllers["statefulset"] = startStatefulSetController
	controllers["cronjob"] = startCronJobController
	controllers["csrsigning"] = startCSRSigningController
	controllers["csrapproving"] = startCSRApprovingController
	controllers["ttl"] = startTTLController
	controllers["bootstrapsigner"] = startBootstrapSignerController
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