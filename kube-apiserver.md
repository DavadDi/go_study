# K8S kube-apiserver 源码层次分析

## 1. Run入口
k8s.io/kubernetes/cmd/kube-apiserver/apiserver.go

```go
func main() {
	// .....
	s := options.NewServerRunOptions()

	if err := app.Run(s, wait.NeverStop); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
```

k8s.io/kubernetes/cmd/kube-apiserver/app/server.go

```go
// Run runs the specified APIServer.  This should never exit.
func Run(runOptions *options.ServerRunOptions, stopCh <-chan struct{}) error {
	// 1. CreateKubeAPIServerConfig 完成配置初始化，
	kubeAPIServerConfig, sharedInformers, insecureServingOptions, err := CreateKubeAPIServerConfig(runOptions)
	
	// 2. 创建 apiExtensionsServer
	apiExtensionsConfig, err := createAPIExtensionsConfig(*kubeAPIServerConfig.GenericConfig, runOptions)
	apiExtensionsServer, err := createAPIExtensionsServer(apiExtensionsConfig, genericapiserver.EmptyDelegate)

	
	// 3. 创建 kubeAPIServer
	kubeAPIServer, err := CreateKubeAPIServer(kubeAPIServerConfig, apiExtensionsServer.GenericAPIServer, sharedInformers)
	
	// 4. 创建 aggregatorServer
	aggregatorConfig, err := createAggregatorConfig(*kubeAPIServerConfig.GenericConfig, runOptions)
	aggregatorServer, err := createAggregatorServer(aggregatorConfig, kubeAPIServer.GenericAPIServer, sharedInformers, apiExtensionsServer.Informers)


	// NonBlockingRun spawns the insecure http server.
	if insecureServingOptions != nil {
		insecureHandlerChain := kubeserver.BuildInsecureHandlerChain(aggregatorServer.GenericAPIServer.UnprotectedHandler(), kubeAPIServerConfig.GenericConfig)
		if err := kubeserver.NonBlockingRun(insecureServingOptions, insecureHandlerChain, stopCh); err != nil {
			return err
		}
	}

	return aggregatorServer.GenericAPIServer.PrepareRun().Run(stopCh)
}
```

### Run 初始化

### 疑问

sharedInformers 这个作用？

k8s.io/kubernetes/cmd/kube-apiserver/app/server.go

```go

func BuildGenericConfig(s *options.ServerRunOptions) (*genericapiserver.Config, informers.SharedInformerFactory, *kubeserver.InsecureServingInfo, error) {
	/// ...
	sharedInformers := informers.NewSharedInformerFactory(client, 10*time.Minute)
	/// ...
}


func NewSharedInformerFactory(client internalclientset.Interface, defaultResync time.Duration) SharedInformerFactory {
	return &sharedInformerFactory{
		client:           client,
		defaultResync:    defaultResync,
		informers:        make(map[reflect.Type]cache.SharedIndexInformer),
		startedInformers: make(map[reflect.Type]bool),
	}
}
```