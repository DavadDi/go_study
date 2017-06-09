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

### 1.3 疑问

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