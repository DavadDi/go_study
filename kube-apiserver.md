# K8S kube-apiserver 源码层次分析

## 1. Master

### 1.1 Run入口
初步代码在 cmd/kube-apiserver 和 pkg/master/master.go

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

### 1.2 CreateKubeAPIServer

k8s.io/kubernetes/pkg/master/master.go  -> k8s.io/kubernetes/vendor/k8s.io/apiserver/pkg/server

```go
// Master contains state for a Kubernetes cluster master/api server.
type Master struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
	ClientCARegistrationHook ClientCARegistrationHook
}
```

```go

// CreateKubeAPIServer creates and wires a workable kube-apiserver
func CreateKubeAPIServer(kubeAPIServerConfig *master.Config, delegateAPIServer genericapiserver.DelegationTarget, sharedInformers informers.SharedInformerFactory) (\*master.Master, error) {
	kubeAPIServer, err := kubeAPIServerConfig.Complete().New(delegateAPIServer)
	// ...

	return kubeAPIServer, nil
}

```

```go
// New returns a new instance of Master from the given config.
// Certain config fields will be set to a default value if unset.
// Certain config fields must be specified, including:
//   KubeletClientConfig
func (c completedConfig) New(delegationTarget genericapiserver.DelegationTarget) (*Master, error) {
	// 创建一个 GenericServer 出来，并设置到 master对象中
	// SkipComplete provides a way to construct a server instance without config completion
	// s := GenericAPIServer
	// GenericConfig -> k8s.io/kubernetes/vendor/k8s.io/apiserver/pkg/server/config.go
	s, err := c.Config.GenericConfig.SkipComplete().New("kube-apiserver", delegationTarget) // completion is done in Complete, no need for a second time

	m := &Master{
		GenericAPIServer: s,
	}

	// install legacy rest storage
	if c.APIResourceConfigSource.AnyResourcesForVersionEnabled(apiv1.SchemeGroupVersion) {
		legacyRESTStorageProvider := corerest.LegacyRESTStorageProvider{
			StorageFactory:       c.StorageFactory,
			ProxyTransport:       c.ProxyTransport,
			KubeletClientConfig:  c.KubeletClientConfig,
			EventTTL:             c.EventTTL,
			ServiceIPRange:       c.ServiceIPRange,
			ServiceNodePortRange: c.ServiceNodePortRange,
			LoopbackClientConfig: c.GenericConfig.LoopbackClientConfig,
		}

		// RESTOptionsGetter is used to construct RESTStorage types via the generic registry.
		m.InstallLegacyAPI(c.Config, c.Config.GenericConfig.RESTOptionsGetter, legacyRESTStorageProvider)
	}

	restStorageProviders := []RESTStorageProvider{
		authenticationrest.RESTStorageProvider{Authenticator: c.GenericConfig.Authenticator},
		// ....
		appsrest.RESTStorageProvider{},
		admissionregistrationrest.RESTStorageProvider{},
	}

	m.InstallAPIs(c.Config.APIResourceConfigSource, c.Config.GenericConfig.RESTOptionsGetter, restStorageProviders...)
	
	// ...

	return m, nil
}
```

### 1.3 总结
	
完成的主要流程：

* 从配置文件完成配置结构的转换
* 创建 Master对象，包括 GenericAPIServer，调用 InstallLegacyAPI 和 InstallAPIs 完成相关API处理加载；
* 创建apiExtensionsServer；创建aggregatorServer
* 完成 insecureServer和aggregatorServer启动

### 1.4 其他

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

## 2. GenericAPIServer

代码主体已经转移到了 k8s.io/apiserver/pkg/server/ 这个pkg

### 2.1 GenericAPIServer结构

GenericAPIServer 中重要的字段

```go
type GenericAPIServer struct {

	// storage contains the RESTful endpoints exposed by this GenericAPIServer
	storage map[string]rest.Storage
	// 未在 GenericAPIServer 函数中看到使用？ 需要进一步确认
	
	// ....
	
	// "Outputs"
	// Handler holdes the handlers being used by this API server
	Handler *APIServerHandler
}
```

### 2.2 GenericAPIServer创建

k8s.io/kubernetes/vendor/k8s.io/apiserver/pkg/server/config.go

```go
// New creates a new server which logically combines the handling chain with the passed server.
// name is used to differentiate for logging.  The handler chain in particular can be difficult as it starts delgating.
func (c completedConfig) New(name string, delegationTarget DelegationTarget) (*GenericAPIServer, error) {
	handlerChainBuilder := func(handler http.Handler) http.Handler {
		return c.BuildHandlerChainFunc(handler, c.Config)
	}

	// NewAPIServerHandler
	apiServerHandler := NewAPIServerHandler(name, c.RequestContextMapper, c.Serializer, handlerChainBuilder, delegationTarget.UnprotectedHandler())

	s := &GenericAPIServer{
		// ...
		Handler: apiServerHandler, // 为http提供服务总的入口， director -> {nogorustful, gorestful}
		// ...
	}

	genericApiServerHookName := "generic-apiserver-start-informers"
	if c.SharedInformerFactory != nil && !s.isHookRegistered(genericApiServerHookName) {
		err := s.AddPostStartHook(genericApiServerHookName, func(context PostStartHookContext) error {
			c.SharedInformerFactory.Start(context.StopCh)
			return nil
		})
	}
	
	// install Index/Swagger/Mertircs/Discorvy 
	installAPI(s, c.Config)
	
	// ...

	return s, nil
}

```

k8s.io/kubernetes/vendor/k8s.io/apiserver/pkg/server/handler.go

```go
func NewAPIServerHandler(name string, contextMapper request.RequestContextMapper, s runtime.NegotiatedSerializer, handlerChainBuilder HandlerChainBuilderFn, notFoundHandler http.Handler) *APIServerHandler {
	nonGoRestfulMux := genericmux.NewPathRecorderMux(name)
	if notFoundHandler != nil {
		nonGoRestfulMux.NotFoundHandler(notFoundHandler)
	}

	gorestfulContainer := restful.NewContainer()
	gorestfulContainer.ServeMux = http.NewServeMux()
	gorestfulContainer.Router(restful.CurlyRouter{}) // e.g. for proxy/{kind}/{name}/{*}
	gorestfulContainer.RecoverHandler(func(panicReason interface{}, httpWriter http.ResponseWriter) {
		logStackOnRecover(s, panicReason, httpWriter)
	})
	
	gorestfulContainer.ServiceErrorHandler(func(serviceErr restful.ServiceError, request *restful.Request, response *restful.Response) {
		ctx, ok := contextMapper.Get(request.Request)
		if !ok {
			responsewriters.InternalError(response.ResponseWriter, request.Request, errors.New("no context found for request"))
		}
		serviceErrorHandler(ctx, s, serviceErr, request, response)
	})

	director := director{
		name:               name,
		goRestfulContainer: gorestfulContainer,
		nonGoRestfulMux:    nonGoRestfulMux,
	}

	return &APIServerHandler{
		FullHandlerChain:   handlerChainBuilder(director),
		GoRestfulContainer: gorestfulContainer,
		NonGoRestfulMux:    nonGoRestfulMux,
		Director:           director,
	}
}



// APIServerHandlers holds the different http.Handlers used by the API server.
// This includes the full handler chain, the director (which chooses between gorestful and nonGoRestful,
// the gorestful handler (used for the API) which falls through to the nonGoRestful handler on unregistered paths,
// and the nonGoRestful handler (which can contain a fallthrough of its own)
// FullHandlerChain -> Director -> {GoRestfulContainer,NonGoRestfulMux} based on inspection of registered web services
type APIServerHandler struct {
	// FullHandlerChain is the one that is eventually served with.  It should include the full filter
	// chain and then call the Director.
	FullHandlerChain http.Handler
	// The registered APIs.  InstallAPIs uses this.  Other servers probably shouldn't access this directly.
	GoRestfulContainer *restful.Container
	// NonGoRestfulMux is the final HTTP handler in the chain.
	// It comes after all filters and the API handling
	// This is where other servers can attach handler to various parts of the chain.
	NonGoRestfulMux *mux.PathRecorderMux

	// Director is here so that we can properly handle fall through and proxy cases.
	// This looks a bit bonkers, but here's what's happening.  We need to have /apis handling registered in gorestful in order to have
	// swagger generated for compatibility.  Doing that with `/apis` as a webservice, means that it forcibly 404s (no defaulting allowed)
	// all requests which are not /apis or /apis/.  We need those calls to fall through behind goresful for proper delegation.  Trying to
	// register for a pattern which includes everything behind it doesn't work because gorestful negotiates for verbs and content encoding
	// and all those things go crazy when gorestful really just needs to pass through.  In addition, openapi enforces unique verb constraints
	// which we don't fit into and it still muddies up swagger.  Trying to switch the webservices into a route doesn't work because the
	//  containing webservice faces all the same problems listed above.
	// This leads to the crazy thing done here.  Our mux does what we need, so we'll place it in front of gorestful.  It will introspect to
	// decide if the the route is likely to be handled by goresful and route there if needed.  Otherwise, it goes to PostGoRestful mux in
	// order to handle "normal" paths and delegation. Hopefully no API consumers will ever have to deal with this level of detail.  I think
	// we should consider completely removing gorestful.
	// Other servers should only use this opaquely to delegate to an API server.
	Director http.Handler
}

```


### 2.3 Run

```go
func Run(){
	// ...
	aggregatorServer.GenericAPIServer.PrepareRun().Run(stopCh)
}

type preparedGenericAPIServer struct {
	*GenericAPIServer
}


// Run spawns the secure http server. It only returns if stopCh is closed
// or the secure port cannot be listened on initially.
func (s preparedGenericAPIServer) Run(stopCh <-chan struct{}) error {
	err := s.NonBlockingRun(stopCh)
	//...
}

func (s preparedGenericAPIServer) NonBlockingRun(stopCh <-chan struct{}) error {
	// Use an internal stop channel to allow cleanup of the listeners on error.
	internalStopCh := make(chan struct{})

	if s.SecureServingInfo != nil && s.Handler != nil {
		if err := s.serveSecurely(internalStopCh); err != nil 
		//....
}


// serveSecurely runs the secure http server. It fails only if certificates cannot
// be loaded or the initial listen call fails. The actual server loop (stoppable by closing
// stopCh) runs in a go routine, i.e. serveSecurely does not block.
func (s *GenericAPIServer) serveSecurely(stopCh <-chan struct{}) error {
	secureServer := &http.Server{
		Addr:           s.SecureServingInfo.BindAddress,
		Handler:        s.Handler, // GenericAPIServer { Handler *APIServerHandler }
		MaxHeaderBytes: 1 << 20,
	
	    // ...
	    
	   s.effectiveSecurePort, err = RunServer(secureServer, s.SecureServingInfo.BindNetwork, stopCh)
	   
	   // ...
}


// RunServer listens on the given port, then spawns a go-routine continuously serving
// until the stopCh is closed. The port is returned. This function does not block.
func RunServer(server *http.Server, network string, stopCh <-chan struct{}) (int, error) {
		// ...
		err := server.Serve(listener)
		//....
}
```


## 3. InstallLegacyAPI 流程

### 3.1 Master InstallLegacyAPI
LegacyAPI 主要是对于 APIServer Core核心部分的相关处理；registry相关的代码目录为 **k8s.io/kubernetes/pkg/registry/core**


```go
func (c completedConfig) New(delegationTarget genericapiserver.DelegationTarget) (*Master, error) {	// ...
	// install legacy rest storage
	if c.APIResourceConfigSource.AnyResourcesForVersionEnabled(apiv1.SchemeGroupVersion) {
		legacyRESTStorageProvider := corerest.LegacyRESTStorageProvider{
			StorageFactory:       c.StorageFactory,
			ProxyTransport:       c.ProxyTransport,
			KubeletClientConfig:  c.KubeletClientConfig,
			EventTTL:             c.EventTTL,
			ServiceIPRange:       c.ServiceIPRange,
			ServiceNodePortRange: c.ServiceNodePortRange,
			LoopbackClientConfig: c.GenericConfig.LoopbackClientConfig,
		}

		// RESTOptionsGetter is used to construct RESTStorage types via the generic registry.
		m.InstallLegacyAPI(c.Config, c.Config.GenericConfig.RESTOptionsGetter, legacyRESTStorageProvider)
	}
	// ...
}
```

调用参数： master.Config  master.Config.GenericConfig.RESTOptionsGetter  legacyRESTStorageProvider

```go

RESTOptionsGetter genericregistry.RESTOptionsGetter

// 从后端存储获取保存的相关信息
type RESTOptionsGetter interface {
	GetRESTOptions(resource schema.GroupResource) (RESTOptions, error)
}

type LegacyRESTStorageProvider struct {
	StorageFactory serverstorage.StorageFactory
	// Used for custom proxy dialing, and proxy TLS options
	ProxyTransport      http.RoundTripper
	KubeletClientConfig kubeletclient.KubeletClientConfig
	EventTTL            time.Duration

	// ServiceIPRange is used to build cluster IPs for discovery.
	ServiceIPRange       net.IPNet
	ServiceNodePortRange utilnet.PortRange

	LoopbackClientConfig *restclient.Config
}
```

master.Config.GenericConfig.RESTOptionsGetter 的设置的来源：

```go
func Run(){
	// ...
	kubeAPIServerConfig, sharedInformers, insecureServingOptions, err := CreateKubeAPIServerConfig(runOptions)
	// ...
}
```

Run() -> CreateKubeAPIServerConfig() -> BuildGenericConfig() -> s.Etcd.ApplyWithStorageFactoryTo(storageFactory, genericConfig) -> 其中 s *options.ServerRunOptions，

vendor/k8s.io/apiserver/pkg/server/options/etcd.go
```go
func (s *EtcdOptions) ApplyWithStorageFactoryTo(factory serverstorage.StorageFactory, c *server.Config) error {
	c.RESTOptionsGetter = &storageFactoryRestOptionsFactory{Options: *s, StorageFactory: factory}
	return nil
}
```

 
```go
func (m *Master) InstallLegacyAPI(c *Config, restOptionsGetter generic.RESTOptionsGetter, legacyRESTStorageProvider corerest.LegacyRESTStorageProvider) {
	legacyRESTStorage, apiGroupInfo, err := legacyRESTStorageProvider.NewLegacyRESTStorage(restOptionsGetter)

	// ...
	
	if err := m.GenericAPIServer.InstallLegacyAPIGroup(genericapiserver.DefaultLegacyAPIPrefix, &apiGroupInfo); err != nil {
		glog.Fatalf("Error in registering group versions: %v", err)
	}
}

```

传入参数：准备工作，调用 legacyRESTStorageProvider.NewLegacyRESTStorage 生成相关的数据。

k8s.io/kubernetes/pkg/registry/core/rest/storage_core.go

```go
func (c LegacyRESTStorageProvider) NewLegacyRESTStorage(restOptionsGetter generic.RESTOptionsGetter) (LegacyRESTStorage, genericapiserver.APIGroupInfo, error) {
	apiGroupInfo := genericapiserver.APIGroupInfo{
		GroupMeta:                    *api.Registry.GroupOrDie(api.GroupName),
		VersionedResourcesStorageMap: map[string]map[string]rest.Storage{},
		Scheme:                      api.Scheme, // runtime.NewScheme()
		ParameterCodec:              api.ParameterCodec, // runtime.NewParameterCodec(Scheme) 		NegotiatedSerializer:        api.Codecs, // serializer.NewCodecFactory(Scheme), 
												        // Codecs provides access to encoding and decoding for the scheme

		SubresourceGroupVersionKind: map[string]schema.GroupVersionKind{},
		// type GroupVersionKind struct { Group,Version,Kind  string}
	}

	restStorage := LegacyRESTStorage{}
	
	// ... 实现会在 3.2 章节详细分析

	podStorage := podstore.NewStorage(
		restOptionsGetter,
		nodeStorage.KubeletConnectionInfo,
		c.ProxyTransport,
		podDisruptionClient,
	)
	
	// ...

	controllerStorage := controllerstore.NewStorage(restOptionsGetter)

	serviceRest := service.NewStorage(serviceRegistry, endpointRegistry, ServiceClusterIPAllocator, ServiceNodePortAllocator, c.ProxyTransport)

	restStorageMap := map[string]rest.Storage{
		"pods":             podStorage.Pod,
		"pods/attach":      podStorage.Attach,
		// ...
		"nodes":        nodeStorage.Node,
		"nodes/status": nodeStorage.Status,
		"nodes/proxy":  nodeStorage.Proxy,

		"events": eventStorage,
		// ...
		"configMaps": configMapStorage,
		"componentStatuses": componentstatus.NewStorage(componentStatusStorage{c.StorageFactory}.serversToValidate),
	}

	apiGroupInfo.VersionedResourcesStorageMap["v1"] = restStorageMap
	return restStorage, apiGroupInfo, nil
}
```

相关结构：

```go
// k8s.io/kubernetes/pkg/registry/core/rest/storage_core.go
type LegacyRESTStorage struct {
	ServiceClusterIPAllocator rangeallocation.RangeRegistry
	ServiceNodePortAllocator  rangeallocation.RangeRegistry
}


// k8s.io/kubernetes/vendor/k8s.io/apiserver/pkg/server/genericapiserver.go
// Info about an API group.
type APIGroupInfo struct {
	GroupMeta apimachinery.GroupMeta
	// Info about the resources in this group. Its a map from version to resource to the storage.
	VersionedResourcesStorageMap map[string]map[string]rest.Storage
	
	// OptionsExternalVersion controls the APIVersion used for common objects in the
	// schema like api.Status, api.DeleteOptions, and metav1.ListOptions. Other implementors may
	// define a version "v1beta1" but want to use the Kubernetes "v1" internal objects.
	// If nil, defaults to groupMeta.GroupVersion.
	// TODO: Remove this when https://github.com/kubernetes/kubernetes/issues/19018 is fixed.
	
	OptionsExternalVersion *schema.GroupVersion // Group, Version string
	// MetaGroupVersion defaults to "meta.k8s.io/v1" and is the scheme group version used to decode
	// common API implementations like ListOptions. Future changes will allow this to vary by group
	// version (for when the inevitable meta/v2 group emerges).
	MetaGroupVersion *schema.GroupVersion

	// Scheme includes all of the types used by this group and how to convert between them (or
	// to convert objects from outside of this group that are accepted in this API).
	// TODO: replace with interfaces
	Scheme *runtime.Scheme
	
	// NegotiatedSerializer controls how this group encodes and decodes data
	NegotiatedSerializer runtime.NegotiatedSerializer
	
	// ParameterCodec performs conversions for query parameters passed to API calls
	ParameterCodec runtime.ParameterCodec

	// SubresourceGroupVersionKind contains the GroupVersionKind overrides for each subresource that is
	// accessible from this API group version. The GroupVersionKind is that of the external version of
	// the subresource. The key of this map should be the path of the subresource. The keys here should
	// match the keys in the Storage map above for subresources.
	SubresourceGroupVersionKind map[string]schema.GroupVersionKind
}
```


codec:
k8s.io/kubernetes/vendor/k8s.io/apimachinery/pkg/runtime/serializer/codec_factory.go

```go
var Codecs = serializer.NewCodecFactory(Scheme)

func NewCodecFactory(scheme *runtime.Scheme) CodecFactory {
	serializers := newSerializersForScheme(scheme, json.DefaultMetaFactory)
	return newCodecFactory(scheme, serializers)
}
```

### 3.2 Storage探究

该章节为Storage实现的分析，暂时可以先跳过，等最后回来进行回溯分析。

```go
// -----------------------
	podStorage := podstore.NewStorage(
		restOptionsGetter,
		nodeStorage.KubeletConnectionInfo,
		c.ProxyTransport,
		podDisruptionClient,
	)
//----------------------

type Store struct {
	// ....
	
	// Storage is the interface for the underlying storage for the resource.
	Storage storage.Interface
	
	// ...
}



// NewStorage returns a RESTStorage object that will work against pods.
func NewStorage(optsGetter generic.RESTOptionsGetter, k client.ConnectionInfoGetter, proxyTransport http.RoundTripper, podDisruptionBudgetClient policyclient.PodDisruptionBudgetsGetter) PodStorage {
	store := &genericregistry.Store{
		Copier:            api.Scheme,
		NewFunc:           func() runtime.Object { return &api.Pod{} },
		NewListFunc:       func() runtime.Object { return &api.PodList{} },
		PredicateFunc:     pod.MatchPod,
		QualifiedResource: api.Resource("pods"),
		WatchCacheSize:    cachesize.GetWatchCacheSizeByResource("pods"),

		CreateStrategy:      pod.Strategy,
		UpdateStrategy:      pod.Strategy,
		DeleteStrategy:      pod.Strategy,
		ReturnDeletedObject: true,
	}
	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: pod.GetAttrs, TriggerFunc: pod.NodeNameTriggerFunc}
	if err := store.CompleteWithOptions(options); err != nil {
		panic(err) // TODO: Propagate error up
	}

	statusStore := *store
	statusStore.UpdateStrategy = pod.StatusStrategy

	return PodStorage{
		Pod:         &REST{store, proxyTransport},
		Binding:     &BindingREST{store: store},
		Eviction:    newEvictionStorage(store, podDisruptionBudgetClient),
		Status:      &StatusREST{store: &statusStore},
		Log:         &podrest.LogREST{Store: store, KubeletConn: k},
		Proxy:       &podrest.ProxyREST{Store: store, ProxyTransport: proxyTransport},
		Exec:        &podrest.ExecREST{Store: store, KubeletConn: k},
		Attach:      &podrest.AttachREST{Store: store, KubeletConn: k},
		PortForward: &podrest.PortForwardREST{Store: store, KubeletConn: k},
	}
}

// CompleteWithOptions updates the store with the provided options and
// defaults common fields.
func (e *Store) CompleteWithOptions(options *generic.StoreOptions) error {
	// ..
	
	// 生成opt选项
	// options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: pod.GetAttrs, TriggerFunc: pod.NodeNameTriggerFunc}
	// options.RESTOptions.GetRESTOptions = optsGetter(restOptionsGetter).GetRESTOptions
	// restOptionsGetter (3.1) = &storageFactoryRestOptionsFactory{Options: *s, StorageFactory: factory}
	// opts == storageFactoryRestOptionsFactory::GetRESTOptions() -> generic.RESTOptions
	
	opts, err := options.RESTOptions.GetRESTOptions(e.QualifiedResource) // type: generic.RESTOptions
	
	// ....
	
	/* 	
	opt = ret := generic.RESTOptions{
		StorageConfig:           storageConfig,
		Decorator:               generic.UndecoratedStorage,
		DeleteCollectionWorkers: f.Options.DeleteCollectionWorkers,
		EnableGarbageCollection: f.Options.EnableGarbageCollection,
		ResourcePrefix:          f.StorageFactory.ResourcePrefix(resource),
	}
	*/
	if e.Storage == nil {                          
		e.Storage, e.DestroyFunc = opts.Decorator( // generic.RESTOptions.Decorator
			*server.Config)                                			e.Copier,
			opts.StorageConfig, 
			e.WatchCacheSize,
			e.NewFunc(),
			prefix,
			keyFunc,
			e.NewListFunc,
			options.AttrFunc,
			triggerFunc,
		)
	}

	return nil
}

// vendor/k8s.io/apiserver/pkg/server/options/etcd.go
func (f *storageFactoryRestOptionsFactory) GetRESTOptions(resource schema.GroupResource) (generic.RESTOptions, error) {

   // k8s.io/kubernetes/vendor/k8s.io/apiserver/pkg/server/storage/storage_factory.go
   // StorageFactory = DefaultStorageFactory
   
	storageConfig, err := f.StorageFactory.NewConfig(resource)
	if err != nil {
		return generic.RESTOptions{}, fmt.Errorf("unable to find storage destination for %v, due to %v", resource, err.Error())
	}

	ret := generic.RESTOptions{
		StorageConfig:           storageConfig,
		Decorator:               generic.UndecoratedStorage,     // opts.Decorator ---------
		DeleteCollectionWorkers: f.Options.DeleteCollectionWorkers,
		EnableGarbageCollection: f.Options.EnableGarbageCollection,
		ResourcePrefix:          f.StorageFactory.ResourcePrefix(resource),
	}
	if f.Options.EnableWatchCache {
		ret.Decorator = genericregistry.StorageWithCacher(f.Options.DefaultWatchCacheSize)
	}

	return ret, nil
}

// k8s.io/kubernetes/vendor/k8s.io/apiserver/pkg/registry/generic/storage_decorator.go

// UndecoratedStorage returns the given a new storage from the given config
// without any decoration.
func UndecoratedStorage(
	copier runtime.ObjectCopier,
	config *storagebackend.Config,
	capacity *int,
	objectType runtime.Object,
	resourcePrefix string,
	keyFunc func(obj runtime.Object) (string, error),
	newListFunc func() runtime.Object,
	getAttrsFunc storage.AttrFunc,
	trigger storage.TriggerPublisherFunc) (storage.Interface, factory.DestroyFunc) {
	
	return NewRawStorage(config)
	
}

// NewRawStorage creates the low level kv storage. This is a work-around for current
// two layer of same storage interface.
func NewRawStorage(config *storagebackend.Config) (storage.Interface, factory.DestroyFunc) {
	s, d, err := factory.Create(*config)
	if err != nil {
		glog.Fatalf("Unable to create storage backend: config (%v), err (%v)", config, err)
	}
	return s, d
}

// vendor/k8s.io/apiserver/pkg/storage/storagebackend/factory/factory.go
// Create creates a storage backend based on given config.
func Create(c storagebackend.Config) (storage.Interface, DestroyFunc, error) {
	switch c.Type {
	case storagebackend.StorageTypeETCD2:
		return newETCD2Storage(c)
	case storagebackend.StorageTypeUnset, storagebackend.StorageTypeETCD3:
		// TODO: We have the following features to implement:
		// - Support secure connection by using key, cert, and CA files.
		// - Honor "https" scheme to support secure connection in gRPC.
		// - Support non-quorum read.
		return newETCD3Storage(c)
	default:
		return nil, nil, fmt.Errorf("unknown storage type: %s", c.Type)
	}
}
```

### 3.3 GenericAPIServer InstallLegacyAPIGroup

```go
func (m *Master) InstallLegacyAPI(){
	
	//...
	// DefaultLegacyAPIPrefix = "/api"
	m.GenericAPIServer.InstallLegacyAPIGroup(genericapiserver.DefaultLegacyAPIPrefix, &apiGroupInfo); 
	// ...
}
```

```go
func (s *GenericAPIServer) InstallLegacyAPIGroup(apiPrefix string, apiGroupInfo *APIGroupInfo) error {
	if err := s.installAPIResources(apiPrefix, apiGroupInfo); err != nil {
		return err
	}

	// setup discovery
	apiVersions := []string{}
	for _, groupVersion := range apiGroupInfo.GroupMeta.GroupVersions {
		apiVersions = append(apiVersions, groupVersion.Version)
	}
	// Install the version handler.
	// Add a handler at /<apiPrefix> to enumerate the supported api versions.
	s.Handler.GoRestfulContainer.Add(discovery.NewLegacyRootAPIHandler(s.discoveryAddresses, s.Serializer, apiPrefix, apiVersions, s.requestContextMapper).WebService())
	return nil
}
```

```go
// installAPIResources is a private method for installing the REST storage backing each api groupversionresource
func (s *GenericAPIServer) installAPIResources(apiPrefix string, apiGroupInfo *APIGroupInfo) error {
	for _, groupVersion := range apiGroupInfo.GroupMeta.GroupVersions {
		// ! 从 apiGroupInfo 得到 apiGroupVersion 对象
		apiGroupVersion := s.getAPIGroupVersion(apiGroupInfo, groupVersion, apiPrefix)
		if apiGroupInfo.OptionsExternalVersion != nil {
			apiGroupVersion.OptionsExternalVersion = apiGroupInfo.OptionsExternalVersion
		}

		if err := apiGroupVersion.InstallREST(s.Handler.GoRestfulContainer); err != nil {
			// ...
		}
	}

	return nil
}
```

getAPIGroupVersion: apiGroupInfo -> groupVersion

```go
func (s *GenericAPIServer) getAPIGroupVersion(apiGroupInfo *APIGroupInfo, groupVersion schema.GroupVersion, apiPrefix string) *genericapi.APIGroupVersion {
	storage := make(map[string]rest.Storage)
	for k, v := range apiGroupInfo.VersionedResourcesStorageMap[groupVersion.Version] {
		storage[strings.ToLower(k)] = v
	}
	version := s.newAPIGroupVersion(apiGroupInfo, groupVersion)
	version.Root = apiPrefix
	version.Storage = storage
	return version
}

func (s *GenericAPIServer) newAPIGroupVersion(apiGroupInfo *APIGroupInfo, groupVersion schema.GroupVersion) *genericapi.APIGroupVersion {
	return &genericapi.APIGroupVersion{
		GroupVersion:     groupVersion,
		MetaGroupVersion: apiGroupInfo.MetaGroupVersion,

		ParameterCodec:  apiGroupInfo.ParameterCodec,
		Serializer:      apiGroupInfo.NegotiatedSerializer,
		Creater:         apiGroupInfo.Scheme,
		Convertor:       apiGroupInfo.Scheme,
		UnsafeConvertor: runtime.UnsafeObjectConvertor(apiGroupInfo.Scheme),
		Copier:          apiGroupInfo.Scheme,
		Defaulter:       apiGroupInfo.Scheme,
		Typer:           apiGroupInfo.Scheme,
		SubresourceGroupVersionKind: apiGroupInfo.SubresourceGroupVersionKind,
		Linker: apiGroupInfo.GroupMeta.SelfLinker,
		Mapper: apiGroupInfo.GroupMeta.RESTMapper,

		Admit:             s.admissionControl,
		Context:           s.RequestContextMapper(),
		MinRequestTimeout: s.minRequestTimeout,
	}
}
```

### 3.3 APIGroupVersion InstallREST

k8s.io/kubernetes/vendor/k8s.io/apiserver/pkg/endpoints/groupversion.go

```go
// InstallREST registers the REST handlers (storage, watch, proxy and redirect) into a restful Container.
// It is expected that the provided path root prefix will serve all operations. Root MUST NOT end
// in a slash.
func (g *APIGroupVersion) InstallREST(container *restful.Container) error {
	installer := g.newInstaller()
	ws := installer.NewWebService()
	apiResources, registrationErrors := installer.Install(ws) // Install 为包装函数
	lister := g.ResourceLister
	if lister == nil {
		lister = staticLister{apiResources}
	}
	versionDiscoveryHandler := discovery.NewAPIVersionHandler(g.Serializer, g.GroupVersion, lister, g.Context)
	versionDiscoveryHandler.AddToWebService(ws)
	container.Add(ws)
	return utilerrors.NewAggregate(registrationErrors)
}
```

### 3.4 Installer 装配车间

k8s.io/kubernetes/vendor/k8s.io/apiserver/pkg/endpoints/installer.go

生成 restful.WebService 对象
```go
// NewWebService creates a new restful webservice with the api installer's prefix and version.
func (a *APIInstaller) NewWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path(a.prefix)
	// a.prefix contains "prefix/group/version"
	ws.Doc("API at " + a.prefix)
	// Backwards compatibility, we accepted objects with empty content-type at V1.
	// If we stop using go-restful, we can default empty content-type to application/json on an
	// endpoint by endpoint basis
	ws.Consumes("*/*")
	mediaTypes, streamMediaTypes := negotiation.MediaTypesForSerializer(a.group.Serializer)
	ws.Produces(append(mediaTypes, streamMediaTypes...)...)
	ws.ApiVersion(a.group.GroupVersion.String())

	return ws
}
```

```go
// Installs handlers for API resources.
func (a *APIInstaller) Install(ws *restful.WebService) (apiResources []metav1.APIResource, errors []error) {
	errors = make([]error, 0)

	proxyHandler := (&handlers.ProxyHandler{
		Prefix:     a.prefix + "/proxy/",
		Storage:    a.group.Storage,
		Serializer: a.group.Serializer,
		Mapper:     a.group.Context,
	})

	// Register the paths in a deterministic (sorted) order to get a deterministic swagger spec.
	paths := make([]string, len(a.group.Storage))
	var i int = 0
	for path := range a.group.Storage {
		paths[i] = path
		i++
	}
	sort.Strings(paths)
	for _, path := range paths {
		// 循环调用 registerResourceHandlers 完成注册
		apiResource, err := a.registerResourceHandlers(path, a.group.Storage[path], ws, proxyHandler)
		if err != nil {
			errors = append(errors, fmt.Errorf("error in registering resource: %s, %v", path, err))
		}
		if apiResource != nil {
			apiResources = append(apiResources, *apiResource)
		}
	}
	return apiResources, errors
}
```



```go
func (a *APIInstaller) registerResourceHandlers(path string, storage rest.Storage, ws *restful.WebService, proxyHandler http.Handler) (*metav1.APIResource, error){
	// 最终装载车间，各式各样的判断和加载
	// ...
	
		switch action.Verb {
		case "GET": // Get a resource.
			var handler restful.RouteFunction
			if isGetterWithOptions {
				handler = restfulGetResourceWithOptions(getterWithOptions, reqScope, hasSubresource)
			} else {
				handler = restfulGetResource(getter, exporter, reqScope)
			}
			handler = metrics.InstrumentRouteFunc(action.Verb, resource, subresource, handler)
			doc := "read the specified " + kind
			if hasSubresource {
				doc = "read " + subresource + " of the specified " + kind
			}
			route := ws.GET(action.Path).To(handler).
				Doc(doc).
				Param(ws.QueryParameter("pretty", "If 'true', then the output is pretty printed.")).
				Operation("read"+namespaced+kind+strings.Title(subresource)+operationSuffix).
				Produces(append(storageMeta.ProducesMIMETypes(action.Verb), mediaTypes...)...).
				Returns(http.StatusOK, "OK", versionedObject).
				Writes(versionedObject)
			if isGetterWithOptions {
				if err := addObjectParams(ws, route, versionedGetOptions); err != nil {
					return nil, err
				}
			}
			if isExporter {
				if err := addObjectParams(ws, route, versionedExportOptions); err != nil {
					return nil, err
				}
			}
			addParams(route, action.Params)
			routes = append(routes, route)
		case "LIST": // List all resources of a kind.
		
		// ...
}
```

```go
func restfulGetResource(r rest.Getter, e rest.Exporter, scope handlers.RequestScope) restful.RouteFunction {
	return func(req *restful.Request, res *restful.Response) {
		handlers.GetResource(r, e, scope)(res.ResponseWriter, req.Request)
	}
}

k8s.io/kubernetes/vendor/k8s.io/apiserver/pkg/endpoints/handlers/rest.go

// GetResource returns a function that handles retrieving a single resource from a rest.Storage object.
func GetResource(r rest.Getter, e rest.Exporter, scope RequestScope) http.HandlerFunc {
	return getResourceHandler(scope,
		func(ctx request.Context, name string, req *http.Request, trace *utiltrace.Trace) (runtime.Object, error) {
			// check for export
			options := metav1.GetOptions{}
			if values := req.URL.Query(); len(values) > 0 {
				exports := metav1.ExportOptions{}
				if err := metainternalversion.ParameterCodec.DecodeParameters(values, scope.MetaGroupVersion, &exports); err != nil {
					return nil, err
				}
				if exports.Export {
					if e == nil {
						return nil, errors.NewBadRequest(fmt.Sprintf("export of %q is not supported", scope.Resource.Resource))
					}
					return e.Export(ctx, name, exports)
				}
				if err := metainternalversion.ParameterCodec.DecodeParameters(values, scope.MetaGroupVersion, &options); err != nil {
					return nil, err
				}
			}
			if trace != nil {
				trace.Step("About to Get from storage")
			}
			return r.Get(ctx, name, &options)
		})
}

```

## 4. runtime.Scheme

章节3中的 APIGroupVersion 实现了多版本管理，但为了实现多版本结构体之间的转换，k8s 采用 runtime.Scheme 进行各版本结构体的转换。

```go
// NewScheme creates a new Scheme. This scheme is pluggable by default.
func NewScheme() *Scheme {
	s := &Scheme{
		gvkToType:        map[schema.GroupVersionKind]reflect.Type{},
		typeToGVK:        map[reflect.Type][]schema.GroupVersionKind{},
		unversionedTypes: map[reflect.Type]schema.GroupVersionKind{},
		unversionedKinds: map[string]reflect.Type{},
		cloner:           conversion.NewCloner(),
		fieldLabelConversionFuncs: map[string]map[string]FieldLabelConversionFunc{},
		defaulterFuncs:            map[reflect.Type]func(interface{}){},
	}
	s.converter = conversion.NewConverter(s.nameFunc)

	s.AddConversionFuncs(DefaultEmbeddedConversions()...)

	// Enable map[string][]string conversions by default
	if err := s.AddConversionFuncs(DefaultStringConversions...); err != nil {
		panic(err)
	}
	if err := s.RegisterInputDefaults(&map[string][]string{}, JSONKeyMapper, conversion.AllowDifferentFieldTypeNames|conversion.IgnoreMissingFields); err != nil {
		panic(err)
	}
	if err := s.RegisterInputDefaults(&url.Values{}, JSONKeyMapper, conversion.AllowDifferentFieldTypeNames|conversion.IgnoreMissingFields); err != nil {
		panic(err)
	}
	return s
}
```

```go
// Scheme defines methods for serializing and deserializing API objects, a type
// registry for converting group, version, and kind information to and from Go
// schemas, and mappings between Go schemas of different versions. A scheme is the
// foundation for a versioned API and versioned configuration over time.
//
// In a Scheme, a Type is a particular Go struct, a Version is a point-in-time
// identifier for a particular representation of that Type (typically backwards
// compatible), a Kind is the unique name for that Type within the Version, and a
// Group identifies a set of Versions, Kinds, and Types that evolve over time. An
// Unversioned Type is one that is not yet formally bound to a type and is promised
// to be backwards compatible (effectively a "v1" of a Type that does not expect
// to break in the future).
//
// Schemes are not expected to change at runtime and are only threadsafe after
// registration is complete.
type Scheme struct {
	// versionMap allows one to figure out the go type of an object with
	// the given version and name.
	gvkToType map[schema.GroupVersionKind]reflect.Type

	// typeToGroupVersion allows one to find metadata for a given go object.
	// The reflect.Type we index by should *not* be a pointer.
	typeToGVK map[reflect.Type][]schema.GroupVersionKind

	// unversionedTypes are transformed without conversion in ConvertToVersion.
	unversionedTypes map[reflect.Type]schema.GroupVersionKind

	// unversionedKinds are the names of kinds that can be created in the context of any group
	// or version
	// TODO: resolve the status of unversioned types.
	unversionedKinds map[string]reflect.Type

	// Map from version and resource to the corresponding func to convert
	// resource field labels in that version to internal version.
	fieldLabelConversionFuncs map[string]map[string]FieldLabelConversionFunc

	// defaulterFuncs is an array of interfaces to be called with an object to provide defaulting
	// the provided object must be a pointer.
	defaulterFuncs map[reflect.Type]func(interface{})

	// converter stores all registered conversion functions. It also has
	// default coverting behavior.
	converter *conversion.Converter

	// cloner stores all registered copy functions. It also has default
	// deep copy behavior.
	cloner *conversion.Cloner
}

```


k8s.io/kubernetes/vendor/k8s.io/apimachinery/pkg/runtime/serializer

定义了 En/Decoder 包括 json/protobuf/yaml 等

```go
type Serializer interface {
	Encoder
	Decoder
}
```

5. Storage

k8s.io/kubernetes/pkg/registry/core/rest/storage_core.go

```go
func (c LegacyRESTStorageProvider) NewLegacyRESTStorage(restOptionsGetter generic.RESTOptionsGetter) (LegacyRESTStorage, genericapiserver.APIGroupInfo, error){

	podStorage := podstore.NewStorage(
		restOptionsGetter,
		nodeStorage.KubeletConnectionInfo,
		c.ProxyTransport,
		podDisruptionClient,
	)
	
       // ...
     
   restStorageMap := map[string]rest.Storage{
		"pods":             podStorage.Pod,
		"pods/attach":      podStorage.Attach,
		"pods/status":      podStorage.Status,
		
		// ...

		"nodes":        nodeStorage.Node,
		"nodes/status": nodeStorage.Status,
		"nodes/proxy":  nodeStorage.Proxy,

		// ...
	}
	
	apiGroupInfo.VersionedResourcesStorageMap["v1"] = restStorageMap
	
}
```

```go
// Storage is a generic interface for RESTful storage services.
// Resources which are exported to the RESTful API of apiserver need to implement this interface. It is expected
// that objects may implement any of the below interfaces.
type Storage interface {
	// New returns an empty object that can be used with Create and Update after request data has been put into it.
	// This object must be a pointer type for use with Codec.DecodeInto([]byte, runtime.Object)
	New() runtime.Object
}
```

k8s.io/kubernetes/pkg/registry/core/pod/storage/storage.go

```go
// REST implements a RESTStorage for pods
type REST struct {
	*genericregistry.Store  // struct, k8s.io/kubernetes/vendor/k8s.io/apiserver/pkg/registry/generic/registry/store.go
	proxyTransport http.RoundTripper
}

// NewStorage returns a RESTStorage object that will work against pods.
func NewStorage(optsGetter generic.RESTOptionsGetter, k client.ConnectionInfoGetter, proxyTransport http.RoundTripper, podDisruptionBudgetClient policyclient.PodDisruptionBudgetsGetter) PodStorage {
	store := &genericregistry.Store{
		Copier:            api.Scheme,
		NewFunc:           func() runtime.Object { return &api.Pod{} },
		NewListFunc:       func() runtime.Object { return &api.PodList{} },
		PredicateFunc:     pod.MatchPod,
		QualifiedResource: api.Resource("pods"),
		WatchCacheSize:    cachesize.GetWatchCacheSizeByResource("pods"),

		CreateStrategy:      pod.Strategy,
		UpdateStrategy:      pod.Strategy,
		DeleteStrategy:      pod.Strategy,
		ReturnDeletedObject: true,
	}
	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: pod.GetAttrs, TriggerFunc: pod.NodeNameTriggerFunc}
	if err := store.CompleteWithOptions(options); err != nil {
		panic(err) // TODO: Propagate error up
	}

	statusStore := *store
	statusStore.UpdateStrategy = pod.StatusStrategy

	return PodStorage{
		Pod:         &REST{store, proxyTransport},
		Binding:     &BindingREST{store: store},
		Eviction:    newEvictionStorage(store, podDisruptionBudgetClient),
		Status:      &StatusREST{store: &statusStore},
		Log:         &podrest.LogREST{Store: store, KubeletConn: k},
		Proxy:       &podrest.ProxyREST{Store: store, ProxyTransport: proxyTransport},
		Exec:        &podrest.ExecREST{Store: store, KubeletConn: k},
		Attach:      &podrest.AttachREST{Store: store, KubeletConn: k},
		PortForward: &podrest.PortForwardREST{Store: store, KubeletConn: k},
	}
}

```





