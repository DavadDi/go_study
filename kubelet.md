# kubelet 源码分析

## 1. kubelet 整体流程

k8s.io/kubernetes/cmd/kubelet/kubelet.go

```go
func main() {
	// ...

	if err := app.Run(s, nil); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
```

k8s.io/kubernetes/cmd/kubelet/app/server.go

```go
// Run runs the specified KubeletServer with the given KubeletDeps.  This should never exit.
// The kubeDeps argument may be nil - if so, it is initialized from the settings on KubeletServer.
// Otherwise, the caller is assumed to have set up the KubeletDeps object and a default one will
// not be generated.
func Run(s *options.KubeletServer, kubeDeps *kubelet.KubeletDeps) error {
	if err := run(s, kubeDeps); err != nil {
		return fmt.Errorf("failed to run Kubelet: %v", err)
	}
	return nil
}


func run(s *options.KubeletServer, kubeDeps *kubelet.KubeletDeps) (err error) {
	/*kubeDeps = nil*/
	// ...
	done := make(chan struct{})

  	// ...
	// Validate configuration.
	if err := validateConfig(s); err != nil {
		return err
	}

	nodeName, err := getNodeName(kubeDeps.Cloud, nodeutil.GetHostname(s.HostnameOverride))

	// Setup event recorder if required.
	makeEventRecorder(&s.KubeletConfiguration, kubeDeps, nodeName)

	if kubeDeps.ContainerManager == nil {
         // ...
		kubeDeps.ContainerManager, err = cm.NewContainerManager(
			kubeDeps.Mounter,
			kubeDeps.CAdvisorInterface,
			cm.NodeConfig{
				RuntimeCgroupsName:    s.RuntimeCgroups,
				SystemCgroupsName:     s.SystemCgroups,
				KubeletCgroupsName:    s.KubeletCgroups,
				ContainerRuntime:      s.ContainerRuntime,
				CgroupsPerQOS:         s.CgroupsPerQOS,
				CgroupRoot:            s.CgroupRoot,
				CgroupDriver:          s.CgroupDriver,
				ProtectKernelDefaults: s.ProtectKernelDefaults,
				NodeAllocatableConfig: cm.NodeAllocatableConfig{
					KubeReservedCgroupName:   s.KubeReservedCgroup,
					SystemReservedCgroupName: s.SystemReservedCgroup,
					EnforceNodeAllocatable:   sets.NewString(s.EnforceNodeAllocatable...),
					KubeReserved:             kubeReserved,
					SystemReserved:           systemReserved,
					HardEvictionThresholds:   hardEvictionThresholds,
				},
				ExperimentalQOSReserved: *experimentalQOSReserved,
			},
			s.ExperimentalFailSwapOn,
			kubeDeps.Recorder)
	}
  
  	// ....

	RunKubelet(&s.KubeletFlags, &s.KubeletConfiguration, kubeDeps, s.RunOnce, standaloneMode); 
	
	<-done
	return nil
}
```

```go
func RunKubelet(kubeFlags *options.KubeletFlags, kubeCfg *componentconfig.KubeletConfiguration, kubeDeps *kubelet.KubeletDeps, runOnce bool, standaloneMode bool) error {
	hostname := nodeutil.GetHostname(kubeFlags.HostnameOverride)
	// Query the cloud provider for our node name, default to hostname if kcfg.Cloud == nil
	nodeName, err := getNodeName(kubeDeps.Cloud, hostname)
  
	// Setup event recorder if required.
	makeEventRecorder(kubeCfg, kubeDeps, nodeName)
  
	hostNetworkSources, err := kubetypes.GetValidatedSources(kubeCfg.HostNetworkSources)
	hostPIDSources, err := kubetypes.GetValidatedSources(kubeCfg.HostPIDSources)
	hostIPCSources, err := kubetypes.GetValidatedSources(kubeCfg.HostIPCSources)

	privilegedSources := capabilities.PrivilegedSources{
		HostNetworkSources: hostNetworkSources,
		HostPIDSources:     hostPIDSources,
		HostIPCSources:     hostIPCSources,
	}
  
	capabilities.Setup(kubeCfg.AllowPrivileged, privilegedSources, 0)
	credentialprovider.SetPreferredDockercfgPath(kubeCfg.RootDirectory)

	builder := kubeDeps.Builder
	if builder == nil {
		builder = CreateAndInitKubelet // 重点分析
	}
  
	if kubeDeps.OSInterface == nil {
		kubeDeps.OSInterface = kubecontainer.RealOS{}
	}
  
    // 调用 CreateAndInitKubelet 完成 kubelet.KubeletBootstrap 的创建
	k, err := builder(kubeCfg, kubeDeps, &kubeFlags.ContainerRuntimeOptions, standaloneMode, kubeFlags.HostnameOverride, kubeFlags.NodeIP, kubeFlags.ProviderID)

	podCfg := kubeDeps.PodConfig
  	
  	//...
	startKubelet(k, podCfg, kubeCfg, kubeDeps)
}
```

```go
func startKubelet(k kubelet.KubeletBootstrap, podCfg *config.PodConfig, kubeCfg *componentconfig.KubeletConfiguration, kubeDeps *kubelet.KubeletDeps) {
	// start the kubelet
	go wait.Until(func() { k.Run(podCfg.Updates()) }, 0, wait.NeverStop)

	// start the kubelet server， 主要提供给API Server 访问相关信息
	if kubeCfg.EnableServer {
		go wait.Until(func() {
			k.ListenAndServe(net.ParseIP(kubeCfg.Address), uint(kubeCfg.Port), kubeDeps.TLSOptions, kubeDeps.Auth, kubeCfg.EnableDebuggingHandlers, kubeCfg.EnableContentionProfiling)
		}, 0, wait.NeverStop)
	}
  
	if kubeCfg.ReadOnlyPort > 0 {
		go wait.Until(func() {
			k.ListenAndServeReadOnly(net.ParseIP(kubeCfg.Address), uint(kubeCfg.ReadOnlyPort))
		}, 0, wait.NeverStop)
	}
}
```

创建kubelet并初始化相关工作

```go
func CreateAndInitKubelet(kubeCfg *componentconfig.KubeletConfiguration, kubeDeps *kubelet.KubeletDeps, crOptions *options.ContainerRuntimeOptions, standaloneMode bool, hostnameOverride, nodeIP, providerID string) (k kubelet.KubeletBootstrap, err error) {
	// TODO: block until all sources have delivered at least one update to the channel, or break the sync loop
	// up into "per source" synchronizations
	
	// NewMainKubelet 完成对象的创建，非常长的一个函数
	k, err = kubelet.NewMainKubelet(kubeCfg, kubeDeps, crOptions, standaloneMode, hostnameOverride, nodeIP, providerID)
	if err != nil {
		return nil, err
	}
	// 事件通知
	k.BirthCry()
  
    // 启动垃圾回收相关工作
	k.StartGarbageCollection()

	return k, nil
}
```


## 2. kubelet  NewMainKubelet

k8s.io/kubernetes/pkg/kubelet/kubelet.go

```go
// NewMainKubelet instantiates a new Kubelet object along with all the required internal modules.
// No initialization of Kubelet and its modules should happen here.
func NewMainKubelet(kubeCfg *componentconfig.KubeletConfiguration, kubeDeps *KubeletDeps, crOptions *options.ContainerRuntimeOptions, standaloneMode bool, hostnameOverride, nodeIP, providerID string) (*Kubelet, error){

	if kubeDeps.PodConfig == nil {
		var err error
		kubeDeps.PodConfig, err = makePodSourceConfig(kubeCfg, kubeDeps, nodeName)

	// ...
}
```

```go
// makePodSourceConfig creates a config.PodConfig from the given
// KubeletConfiguration or returns an error.
func makePodSourceConfig(kubeCfg *componentconfig.KubeletConfiguration, kubeDeps *KubeletDeps, nodeName types.NodeName) (*config.PodConfig, error) {
	// ...

	// source of all configuration
	cfg := config.NewPodConfig(config.PodConfigNotificationIncremental, kubeDeps.Recorder)

	// define file config source
	if kubeCfg.PodManifestPath != "" {
		config.NewSourceFile(kubeCfg.PodManifestPath, nodeName, kubeCfg.FileCheckFrequency.Duration, cfg.Channel(kubetypes.FileSource))
	}

	// define url config source
	if kubeCfg.ManifestURL != "" {
		config.NewSourceURL(kubeCfg.ManifestURL, manifestURLHeader, nodeName, kubeCfg.HTTPCheckFrequency.Duration, cfg.Channel(kubetypes.HTTPSource))
	}
  
	if kubeDeps.KubeClient != nil {
		config.NewSourceApiserver(kubeDeps.KubeClient, nodeName, cfg.Channel(kubetypes.ApiserverSource))
	}
  
	return cfg, nil
}
```

## 3. kubelet  Run()

k8s.io/kubernetes/pkg/kubelet/kubelet.go

```go
// Run starts the kubelet reacting to config updates
func (kl *Kubelet) Run(updates <-chan kubetypes.PodUpdate) {
   if kl.logServer == nil {
      kl.logServer = http.StripPrefix("/logs/", http.FileServer(http.Dir("/var/log/")))
   }
   if kl.kubeClient == nil {
      glog.Warning("No api server defined - no node status update will be sent.")
   }

   if err := kl.initializeModules(); err != nil {
      kl.recorder.Eventf(kl.nodeRef, v1.EventTypeWarning, events.KubeletSetupFailed, err.Error())
      glog.Error(err)
      kl.runtimeState.setInitError(err)
   }

   // Start volume manager
   go kl.volumeManager.Run(kl.sourcesReady, wait.NeverStop)

   if kl.kubeClient != nil {
      // Start syncing node status immediately, this may set up things the runtime needs to run.
      go wait.Until(kl.syncNodeStatus, kl.nodeStatusUpdateFrequency, wait.NeverStop)
   }
   go wait.Until(kl.syncNetworkStatus, 30*time.Second, wait.NeverStop)
   go wait.Until(kl.updateRuntimeUp, 5*time.Second, wait.NeverStop)

   // Start loop to sync iptables util rules
   if kl.makeIPTablesUtilChains {
      go wait.Until(kl.syncNetworkUtil, 1*time.Minute, wait.NeverStop)
   }

   // Start a goroutine responsible for killing pods (that are not properly
   // handled by pod workers).
   go wait.Until(kl.podKiller, 1*time.Second, wait.NeverStop)

   // Start gorouting responsible for checking limits in resolv.conf
   if kl.resolverConfig != "" {
      go wait.Until(func() { kl.checkLimitsForResolvConf() }, 30*time.Second, wait.NeverStop)
   }

   // Start component sync loops.
   kl.statusManager.Start()
   kl.probeManager.Start()

   // Start the pod lifecycle event generator.
   kl.pleg.Start()
   kl.syncLoop(updates, kl)
}
```

